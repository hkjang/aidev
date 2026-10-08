package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hkjang/umm/internal/intelligence"
	"github.com/jackc/pgx/v5"
)

// embeddingSettingsTTL keeps the ai_gateway lookup off the hot path. Thirty
// seconds is short enough that an administrator sees a model change take effect
// while they are still on the settings screen.
const embeddingSettingsTTL = 30 * time.Second

// embeddingBatchSize bounds one gateway request. Larger batches mean fewer round
// trips; too large and a single slow note blocks a whole canvas load.
const embeddingBatchSize = 64

// embeddingFallbackRetryInterval turns a remote failure into a short circuit
// breaker. Canvas reads keep using the complete local fallback set during the
// window instead of paying the remote timeout again on every page load. An
// administrator changing the gateway configuration clears the window early.
const embeddingFallbackRetryInterval = 5 * time.Minute

var errEmbeddingProviderChanged = errors.New("embedding provider changed while the request was in flight")

// Decrypter is the subset of cryptoutil.Cipher the store needs to read the
// gateway credential. It stays an interface so the store keeps no hard
// dependency on key management.
type Decrypter interface {
	Decrypt(string) (string, error)
}

type embeddingSettings struct {
	BaseURL        string `json:"base_url"`
	APIKey         string `json:"api_key"`
	TimeoutSeconds int    `json:"timeout_seconds"`
	EmbeddingModel string `json:"embedding_model"`
	// Embeddings may live somewhere other than the chat model. umm's own compose
	// file suggests exactly that — a small embedding server beside umm while a
	// larger model answers questions elsewhere — and until now both had to be the
	// same host.
	EmbeddingBaseURL string `json:"embedding_base_url"`
	EmbeddingAPIKey  string `json:"embedding_api_key"`
	// EmbeddingTimeoutSeconds is separate from the chat timeout because the two
	// wait for different things. A chat model composing a Dream is given
	// minutes; embedding a sentence is a millisecond of work behind a network
	// hop, and a caller waiting on it is either searching or indexing. Sharing
	// one number meant a timeout chosen to be generous to the chat model became
	// the time a person waited to see their own notes.
	EmbeddingTimeoutSeconds int `json:"embedding_timeout_seconds"`
}

// defaultEmbeddingTimeout is what an embedding call gets when nothing is
// configured. Short on purpose: past this the answer is not worth the wait, and
// the local algorithm is standing right there.
const defaultEmbeddingTimeout = 10 * time.Second

// embeddingEndpoint resolves where embeddings are sent and what to authenticate
// with there.
//
// Leaving the embedding address empty keeps today's behaviour exactly: the chat
// gateway's address and key. Setting it changes both, and the chat key is not
// carried over — a key issued for one host is a credential, and sending it to a
// different host because a field was left blank would hand it to whoever runs
// that host. An embedding server with no auth is the ordinary case, so an empty
// embedding key means no Authorization header rather than "reuse the other one".
func (settings embeddingSettings) embeddingEndpoint() (baseURL, apiKey string) {
	return ResolveEmbeddingEndpoint(settings.BaseURL, settings.APIKey,
		settings.EmbeddingBaseURL, settings.EmbeddingAPIKey)
}

// ResolveEmbeddingEndpoint decides where embeddings go and what authenticates
// them there, given the four configured values.
//
// Exported so the connection test in the admin screen answers the same question
// the runtime does. It did not, at first: the test read only the chat address
// and so checked a server embeddings were no longer being sent to. A second copy
// of this rule is how the two come apart, so there is one.
func ResolveEmbeddingEndpoint(baseURL, apiKey, embeddingBaseURL, embeddingAPIKey string) (string, string) {
	if custom := strings.TrimSpace(embeddingBaseURL); custom != "" {
		return custom, strings.TrimSpace(embeddingAPIKey)
	}
	return strings.TrimSpace(baseURL), apiKey
}

type embeddingCache struct {
	mu            sync.Mutex
	provider      intelligence.Provider
	loadedAt      time.Time
	remoteRetryAt time.Time
}

// EmbeddingProvider resolves the active embedding backend. With no configured
// embedding model — the default — this is the offline local algorithm.
func (s *Store) EmbeddingProvider(ctx context.Context) intelligence.Provider {
	s.embeddings.mu.Lock()
	defer s.embeddings.mu.Unlock()
	now := time.Now()
	if !s.embeddings.loadedAt.IsZero() && now.Sub(s.embeddings.loadedAt) < embeddingSettingsTTL {
		return availableEmbeddingProvider(s.embeddings.provider, now, s.embeddings.remoteRetryAt)
	}
	provider := intelligence.Provider{}
	var settings embeddingSettings
	if s.GetSetting(ctx, "ai_gateway", &settings) == nil {
		provider = s.embeddingProviderFromSettings(settings)
	}
	if !sameEmbeddingProvider(s.embeddings.provider, provider) {
		s.embeddings.remoteRetryAt = time.Time{}
	}
	s.embeddings.provider = provider
	s.embeddings.loadedAt = now
	return availableEmbeddingProvider(provider, now, s.embeddings.remoteRetryAt)
}

func (s *Store) embeddingProviderFromSettings(settings embeddingSettings) intelligence.Provider {
	baseURL, key := settings.embeddingEndpoint()
	if strings.TrimSpace(settings.EmbeddingModel) == "" || baseURL == "" {
		return intelligence.Provider{}
	}
	if strings.HasPrefix(key, "enc:") {
		if s.Cipher == nil {
			return intelligence.Provider{}
		}
		plain, err := s.Cipher.Decrypt(strings.TrimPrefix(key, "enc:"))
		if err != nil {
			return intelligence.Provider{}
		}
		key = plain
	}
	// The embedding timeout, not the chat one. Falling back to the chat timeout
	// would reintroduce exactly the coupling this separates: an installation
	// that gave its chat model ninety seconds would give a search the same.
	timeout := time.Duration(settings.EmbeddingTimeoutSeconds) * time.Second
	if settings.EmbeddingTimeoutSeconds <= 0 {
		timeout = defaultEmbeddingTimeout
	}
	return intelligence.Provider{Remote: &intelligence.RemoteConfig{
		BaseURL: baseURL, APIKey: key,
		Model: strings.TrimSpace(settings.EmbeddingModel), Timeout: timeout, SettingsManaged: true,
	}}
}

func availableEmbeddingProvider(provider intelligence.Provider, now, remoteRetryAt time.Time) intelligence.Provider {
	if provider.Algorithm() != intelligence.LocalAlgorithm && now.Before(remoteRetryAt) {
		return intelligence.Provider{}
	}
	return provider
}

func sameEmbeddingProvider(left, right intelligence.Provider) bool {
	if left.Remote == nil || right.Remote == nil {
		return left.Remote == nil && right.Remote == nil
	}
	return left.Remote.BaseURL == right.Remote.BaseURL &&
		left.Remote.APIKey == right.Remote.APIKey &&
		left.Remote.Model == right.Remote.Model &&
		left.Remote.Timeout == right.Remote.Timeout &&
		left.Remote.SettingsManaged == right.Remote.SettingsManaged
}

func (s *Store) deferRemoteEmbeddings(provider intelligence.Provider) {
	if provider.Algorithm() == intelligence.LocalAlgorithm {
		return
	}
	s.embeddings.mu.Lock()
	defer s.embeddings.mu.Unlock()
	// A late response from an old configuration must not suppress a newly
	// configured gateway on this instance.
	if !sameEmbeddingProvider(s.embeddings.provider, provider) {
		return
	}
	s.embeddings.remoteRetryAt = time.Now().Add(embeddingFallbackRetryInterval)
}

// InvalidateEmbeddingProvider drops the cached settings so an administrator's
// change takes effect on the next request instead of after the TTL.
func (s *Store) InvalidateEmbeddingProvider() {
	s.embeddings.mu.Lock()
	s.embeddings.loadedAt = time.Time{}
	s.embeddings.remoteRetryAt = time.Time{}
	s.embeddings.mu.Unlock()
}

type embeddingTarget struct {
	ID      uuid.UUID
	Content string
	Version int
}

// embeddingPolicyLease keeps the policy rows that authorized a remote
// embedding request stable until the external call and vector persistence have
// both finished. It uses the bounded AI lease capacity rather than occupying a
// request-pool connection for a potentially slow gateway.
type embeddingPolicyLease struct {
	tx      pgx.Tx
	release func()
}

func (lease *embeddingPolicyLease) rollback() {
	rollbackCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	_ = lease.tx.Rollback(rollbackCtx)
	cancel()
	lease.release()
}

func (s *Store) beginEmbeddingPolicyLease(ctx context.Context, origin intelligence.Provider) (*embeddingPolicyLease, error) {
	tx, release, err := s.BeginAILease(ctx)
	if err != nil {
		return nil, err
	}
	lease := &embeddingPolicyLease{tx: tx, release: release}
	if err = s.requireCurrentEmbeddingProviderTx(ctx, tx, origin); err != nil {
		lease.rollback()
		return nil, err
	}
	return lease, nil
}

// beginEmbeddingNotesLease performs the final note/space exclusion check and
// holds SHARE locks through dispatch. An exclusion that commits first selects
// the local provider; when the lease wins, the policy update waits until no
// captured note body can still be sent to the old policy's gateway.
func (s *Store) beginEmbeddingNotesLease(ctx context.Context, targets []embeddingTarget, origin intelligence.Provider) (*embeddingPolicyLease, bool, error) {
	lease, err := s.beginEmbeddingPolicyLease(ctx, origin)
	if err != nil {
		return nil, false, err
	}
	ids := make([]uuid.UUID, 0, len(targets))
	expected := make(map[uuid.UUID]struct{}, len(targets))
	for _, target := range targets {
		if _, exists := expected[target.ID]; exists {
			continue
		}
		expected[target.ID] = struct{}{}
		ids = append(ids, target.ID)
	}
	rows, err := lease.tx.Query(ctx, `
		SELECT n.id,n.ai_excluded,sp.ai_excluded
		FROM notes n
		JOIN spaces sp ON sp.id=n.space_id
		WHERE n.id=ANY($1) AND n.deleted_at IS NULL
		ORDER BY sp.id,n.id
		FOR SHARE OF sp,n`, ids)
	if err != nil {
		lease.rollback()
		return nil, false, err
	}
	found := make(map[uuid.UUID]struct{}, len(expected))
	allowed := true
	for rows.Next() {
		var id uuid.UUID
		var noteExcluded, spaceExcluded bool
		if err = rows.Scan(&id, &noteExcluded, &spaceExcluded); err != nil {
			rows.Close()
			lease.rollback()
			return nil, false, err
		}
		found[id] = struct{}{}
		if noteExcluded || spaceExcluded {
			allowed = false
		}
	}
	rowsErr := rows.Err()
	rows.Close()
	if rowsErr != nil {
		lease.rollback()
		return nil, false, rowsErr
	}
	if len(found) != len(expected) {
		allowed = false
	}
	if !allowed {
		lease.rollback()
		return nil, false, nil
	}
	return lease, true, nil
}

// beginEmbeddingSearchLease keeps the configured destination and every row
// that selected a scoped comparison algorithm stable until the caller has
// embedded the query and consumed the search rows.
func (s *Store) beginEmbeddingSearchLease(ctx context.Context, userID uuid.UUID, spaceID *uuid.UUID, origin intelligence.Provider) (*embeddingPolicyLease, bool, error) {
	lease, err := s.beginEmbeddingPolicyLease(ctx, origin)
	if err != nil {
		return nil, false, err
	}
	if spaceID == nil {
		return lease, true, nil
	}
	var ownerID uuid.UUID
	var excluded bool
	if err = lease.tx.QueryRow(ctx, `SELECT owner_id,ai_excluded FROM spaces WHERE id=$1 FOR SHARE`, *spaceID).Scan(&ownerID, &excluded); err != nil {
		lease.rollback()
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, false, nil
		}
		return nil, false, err
	}
	if excluded {
		lease.rollback()
		return nil, false, nil
	}
	if ownerID != userID {
		var memberID uuid.UUID
		if err = lease.tx.QueryRow(ctx, `SELECT user_id FROM space_members WHERE space_id=$1 AND user_id=$2 FOR SHARE`, *spaceID, userID).Scan(&memberID); err != nil {
			lease.rollback()
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, false, nil
			}
			return nil, false, err
		}
		if memberID != userID {
			lease.rollback()
			return nil, false, nil
		}
	}
	rows, err := lease.tx.Query(ctx, `
		SELECT ai_excluded
		FROM notes
		WHERE space_id=$1 AND deleted_at IS NULL
		ORDER BY id
		FOR SHARE`, *spaceID)
	if err != nil {
		lease.rollback()
		return nil, false, err
	}
	allowed := true
	for rows.Next() {
		var noteExcluded bool
		if err = rows.Scan(&noteExcluded); err != nil {
			rows.Close()
			lease.rollback()
			return nil, false, err
		}
		if noteExcluded {
			allowed = false
		}
	}
	rowsErr := rows.Err()
	rows.Close()
	if rowsErr != nil {
		lease.rollback()
		return nil, false, rowsErr
	}
	if !allowed {
		lease.rollback()
		return nil, false, nil
	}
	return lease, true, nil
}

// UpsertEmbedding refreshes a single note's vector after a write.
func (s *Store) UpsertEmbedding(ctx context.Context, noteID uuid.UUID, content string, version int) error {
	provider, err := s.embeddingProviderForNote(ctx, noteID)
	if err != nil {
		return err
	}
	_, remoteFailed, err := s.writeEmbeddingsWithProvider(ctx, []embeddingTarget{{ID: noteID, Content: content, Version: version}}, provider)
	if err == nil && remoteFailed {
		s.deferRemoteEmbeddings(provider)
	}
	return err
}

// embeddingProviderForNote fails closed to local embeddings when either the
// note or its containing space opts out of AI processing. The lookup happens
// before content can be handed to a configured remote provider.
func (s *Store) embeddingProviderForNote(ctx context.Context, noteID uuid.UUID) (intelligence.Provider, error) {
	var excluded bool
	err := s.Pool.QueryRow(ctx, `
		SELECT n.ai_excluded OR sp.ai_excluded
		FROM notes n JOIN spaces sp ON sp.id=n.space_id
		WHERE n.id=$1 AND n.deleted_at IS NULL`, noteID).Scan(&excluded)
	if err != nil {
		return intelligence.Provider{}, fmt.Errorf("read embedding exclusion: %w", err)
	}
	if excluded {
		return intelligence.Provider{}, nil
	}
	return s.EmbeddingProvider(ctx), nil
}

func (s *Store) embeddingProviderForNotes(ctx context.Context, notes []Note) intelligence.Provider {
	spaceSet := make(map[uuid.UUID]struct{}, len(notes))
	for _, note := range notes {
		if note.AIExcluded || note.SpaceID == uuid.Nil {
			return intelligence.Provider{}
		}
		spaceSet[note.SpaceID] = struct{}{}
	}
	spaceIDs := make([]uuid.UUID, 0, len(spaceSet))
	for spaceID := range spaceSet {
		spaceIDs = append(spaceIDs, spaceID)
	}
	var spaceExcluded bool
	var found int
	if err := s.Pool.QueryRow(ctx, `SELECT COALESCE(bool_or(ai_excluded),false),count(*) FROM spaces WHERE id=ANY($1)`, spaceIDs).Scan(&spaceExcluded, &found); err != nil || spaceExcluded || found != len(spaceIDs) {
		return intelligence.Provider{}
	}
	return s.EmbeddingProvider(ctx)
}

// embeddingProviderForSearch keeps a query scoped to a space or note-level AI
// exclusion on the same local vector space as its canvas. Authorization and
// exclusion are resolved together before any query text can be handed to a
// configured remote gateway. Missing, inaccessible, or temporarily unreadable
// spaces also fail closed to local.
func (s *Store) embeddingProviderForSearch(ctx context.Context, userID uuid.UUID, spaceID *uuid.UUID) intelligence.Provider {
	if spaceID == nil {
		return s.EmbeddingProvider(ctx)
	}
	var excluded bool
	err := s.Pool.QueryRow(ctx, `
		SELECT sp.ai_excluded OR EXISTS(
		  SELECT 1 FROM notes n
		  WHERE n.space_id=sp.id AND n.deleted_at IS NULL AND n.ai_excluded)
		FROM spaces sp
		WHERE sp.id=$1
		  AND (sp.owner_id=$2 OR EXISTS(
		    SELECT 1 FROM space_members sm WHERE sm.space_id=sp.id AND sm.user_id=$2))`, *spaceID, userID).Scan(&excluded)
	if err != nil || excluded {
		return intelligence.Provider{}
	}
	return s.EmbeddingProvider(ctx)
}

// ensureEmbeddings brings a batch of notes up to date. Before v0.8.0 this issued
// one no-op UPDATE per note on every canvas load; now it asks the database which
// vectors are actually stale and embeds only those.
func (s *Store) ensureEmbeddings(ctx context.Context, notes []Note) {
	if len(notes) == 0 {
		return
	}
	provider := s.embeddingProviderForNotes(ctx, notes)
	algorithm := provider.Algorithm()
	ids := make([]uuid.UUID, len(notes))
	for i, n := range notes {
		ids[i] = n.ID
	}
	current := map[uuid.UUID]int{}
	rows, err := s.Pool.Query(ctx, `SELECT note_id,content_version FROM note_embeddings WHERE note_id=ANY($1) AND algorithm=$2`, ids, algorithm)
	if err == nil {
		for rows.Next() {
			var id uuid.UUID
			var version int
			if rows.Scan(&id, &version) == nil {
				current[id] = version
			}
		}
		rows.Close()
	}
	stale := make([]embeddingTarget, 0, len(notes))
	for _, n := range notes {
		if version, ok := current[n.ID]; ok && version >= n.Version {
			continue
		}
		stale = append(stale, embeddingTarget{ID: n.ID, Content: n.Content, Version: n.Version})
	}
	_, remoteFailed, writeErr := s.writeEmbeddingsWithProvider(ctx, stale, provider)
	if writeErr != nil {
		return
	}
	if remoteFailed {
		s.deferRemoteEmbeddings(provider)
		// A gateway outage can leave older notes with remote vectors and newer
		// notes with local fallbacks. Rewrite the whole comparison set locally so
		// Canvas counts, related notes and clusters never mix or omit vector spaces.
		all := make([]embeddingTarget, len(notes))
		for index, note := range notes {
			all[index] = embeddingTarget{ID: note.ID, Content: note.Content, Version: note.Version}
		}
		_, _, _ = s.writeEmbeddings(ctx, all, intelligence.Provider{}, provider)
	}
}

func (s *Store) writeEmbeddingsWithProvider(ctx context.Context, targets []embeddingTarget, provider intelligence.Provider) (string, bool, error) {
	return s.writeEmbeddings(ctx, targets, provider, provider)
}

// writeEmbeddings lets a caller force the embedding algorithm while retaining
// the settings-backed provider that authorized the overall operation. The
// outage-wide local normalization pass uses a local active provider with its
// original remote provider as origin, so an intervening gateway change fences
// the rewrite just like the first fallback batch.
func (s *Store) writeEmbeddings(ctx context.Context, targets []embeddingTarget, provider, origin intelligence.Provider) (string, bool, error) {
	if len(targets) == 0 {
		return provider.Algorithm(), false, nil
	}
	actualAlgorithm := provider.Algorithm()
	activeProvider := provider
	persistenceOrigin := origin
	remoteFailed := false
	if provider.Algorithm() != intelligence.LocalAlgorithm {
		lease, allowed, err := s.beginEmbeddingNotesLease(ctx, targets, origin)
		if err != nil {
			return actualAlgorithm, false, err
		}
		if allowed {
			defer lease.rollback()
		} else {
			// Exclusion that committed before the final lease wins. Local vectors
			// remain useful, but the remote circuit breaker must not be opened for
			// a deliberate policy fallback.
			activeProvider = intelligence.Provider{}
			persistenceOrigin = intelligence.Provider{}
			actualAlgorithm = intelligence.LocalAlgorithm
		}
	}
	for start := 0; start < len(targets); start += embeddingBatchSize {
		batch := targets[start:min(start+embeddingBatchSize, len(targets))]
		texts := make([]string, len(batch))
		for i, target := range batch {
			texts[i] = target.Content
		}
		requestedProvider := activeProvider
		vectors, algorithm := requestedProvider.Embed(ctx, texts)
		model := requestedProvider.Model()
		if algorithm == intelligence.LocalAlgorithm {
			actualAlgorithm = intelligence.LocalAlgorithm
			model = ""
			if requestedProvider.Algorithm() != intelligence.LocalAlgorithm {
				remoteFailed = true
			}
			// One failed remote batch is enough evidence for this write. Finish
			// the remaining batches locally instead of repeating the full timeout.
			activeProvider = intelligence.Provider{}
		}
		if err := s.persistEmbeddingBatch(ctx, batch, vectors, algorithm, model, persistenceOrigin); err != nil {
			return actualAlgorithm, remoteFailed, err
		}
	}
	return actualAlgorithm, remoteFailed, nil
}

// requireCurrentEmbeddingProviderTx locks the settings generation that selected
// a managed remote provider and rejects work captured under an older endpoint,
// credential, model, or timeout before it can be dispatched or persisted.
func (s *Store) requireCurrentEmbeddingProviderTx(ctx context.Context, tx pgx.Tx, origin intelligence.Provider) error {
	if origin.Remote == nil || !origin.Remote.SettingsManaged {
		return nil
	}
	var raw []byte
	if err := tx.QueryRow(ctx, `SELECT value FROM app_settings WHERE key='ai_gateway' FOR SHARE`).Scan(&raw); err != nil {
		return err
	}
	var settings embeddingSettings
	if err := json.Unmarshal(raw, &settings); err != nil {
		return err
	}
	current := s.embeddingProviderFromSettings(settings)
	if !sameEmbeddingProvider(current, origin) {
		s.InvalidateEmbeddingProvider()
		return errEmbeddingProviderChanged
	}
	return nil
}

// persistEmbeddingBatch fences remote results against the current settings row.
// The shared lock and writes live in one transaction: either an old request
// commits before the administrator's settings update, or it observes the new
// vector-space identifier and is discarded. It can never commit afterward.
func (s *Store) persistEmbeddingBatch(ctx context.Context, batch []embeddingTarget, vectors [][]float32, algorithm, model string, origin intelligence.Provider) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err = s.requireCurrentEmbeddingProviderTx(ctx, tx, origin); err != nil {
		return err
	}
	for i, target := range batch {
		// A newer content version always wins. At the same version, a current
		// provider may replace a different vector space, while an older content
		// version can never win merely because its algorithm differs.
		if _, err = tx.Exec(ctx, `
			INSERT INTO note_embeddings(note_id,algorithm,model,dimensions,vector,content_version,updated_at)
			VALUES($1,$2,$3,$4,$5,$6,now())
			ON CONFLICT(note_id) DO UPDATE SET
			  algorithm=EXCLUDED.algorithm,model=EXCLUDED.model,dimensions=EXCLUDED.dimensions,
			  vector=EXCLUDED.vector,content_version=EXCLUDED.content_version,updated_at=now()
			WHERE note_embeddings.content_version<EXCLUDED.content_version
			   OR (note_embeddings.content_version=EXCLUDED.content_version
			       AND note_embeddings.algorithm<>EXCLUDED.algorithm)`,
			target.ID, algorithm, model, len(vectors[i]), vectors[i], target.Version); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// loadEmbeddings chooses one current algorithm for the entire comparison set.
// The most complete set wins, with the configured provider breaking ties. This
// lets an outage-wide local fallback remain usable without ever mixing it with
// remote vectors.
func (s *Store) loadEmbeddings(ctx context.Context, notes []Note) map[uuid.UUID][]float32 {
	if len(notes) == 0 {
		return map[uuid.UUID][]float32{}
	}
	ids := make([]uuid.UUID, len(notes))
	versions := make(map[uuid.UUID]int, len(notes))
	for i, n := range notes {
		ids[i] = n.ID
		versions[n.ID] = n.Version
	}
	rows, err := s.Pool.Query(ctx, `SELECT note_id,algorithm,vector,content_version FROM note_embeddings WHERE note_id=ANY($1)`, ids)
	if err != nil {
		return map[uuid.UUID][]float32{}
	}
	defer rows.Close()
	byAlgorithm := map[string]map[uuid.UUID][]float32{}
	for rows.Next() {
		var id uuid.UUID
		var algorithm string
		var vector []float32
		var contentVersion int
		if rows.Scan(&id, &algorithm, &vector, &contentVersion) == nil && contentVersion >= versions[id] {
			if byAlgorithm[algorithm] == nil {
				byAlgorithm[algorithm] = map[uuid.UUID][]float32{}
			}
			byAlgorithm[algorithm][id] = vector
		}
	}
	preferred := s.EmbeddingProvider(ctx).Algorithm()
	chosen := ""
	best := -1
	for algorithm, vectors := range byAlgorithm {
		count := len(vectors)
		if count > best || (count == best && algorithm == preferred) || (count == best && chosen != preferred && algorithm < chosen) {
			chosen, best = algorithm, count
		}
	}
	if chosen == "" {
		return map[uuid.UUID][]float32{}
	}
	return byAlgorithm[chosen]
}

// EmbedQuery vectorises a search query and reports the algorithm that actually
// produced it. A remote provider can fall back locally, so callers must use the
// returned label rather than the provider's configured label when selecting
// stored vectors.
func (s *Store) EmbedQuery(ctx context.Context, query string) ([]float32, string) {
	provider := s.EmbeddingProvider(ctx)
	vector, algorithm, lease := s.embedQueryWithPolicy(ctx, uuid.Nil, nil, query, provider)
	if lease != nil {
		defer lease.rollback()
	}
	return vector, algorithm
}

func (s *Store) embedQueryWithPolicy(ctx context.Context, userID uuid.UUID, spaceID *uuid.UUID, query string, provider intelligence.Provider) ([]float32, string, *embeddingPolicyLease) {
	if provider.Algorithm() == intelligence.LocalAlgorithm {
		vector, algorithm := s.embedQueryWithProvider(ctx, query, provider)
		return vector, algorithm, nil
	}
	// An unmanaged provider exists only in tests or specialized embedding
	// callers. A scoped search still needs its space policy lease; a global one
	// has no database-managed destination or exclusion row to fence.
	if provider.Remote != nil && !provider.Remote.SettingsManaged && spaceID == nil {
		vector, algorithm := s.embedQueryWithProvider(ctx, query, provider)
		return vector, algorithm, nil
	}
	lease, allowed, err := s.beginEmbeddingSearchLease(ctx, userID, spaceID, provider)
	if err != nil || !allowed {
		vector, algorithm := s.embedQueryWithProvider(ctx, query, intelligence.Provider{})
		return vector, algorithm, nil
	}
	vector, algorithm := s.embedQueryWithProvider(ctx, query, provider)
	return vector, algorithm, lease
}

func (s *Store) embedQueryWithProvider(ctx context.Context, query string, provider intelligence.Provider) ([]float32, string) {
	vectors, algorithm := provider.Embed(ctx, []string{query})
	if provider.Algorithm() != intelligence.LocalAlgorithm && algorithm == intelligence.LocalAlgorithm {
		s.deferRemoteEmbeddings(provider)
	}
	if len(vectors) == 0 {
		return nil, algorithm
	}
	return vectors[0], algorithm
}
