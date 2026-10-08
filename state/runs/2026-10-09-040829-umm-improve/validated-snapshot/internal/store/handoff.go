package store

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

/*
Handing a space to another service.

A claim is the whole of what one service shows another: a random token bound
to one document and one person's right to read it, good for five minutes and
for one collection. No service holds a credential for umm; whoever brings the
token gets the document, once, and the token is spent.

The document is rendered when the claim is issued and kept with it, so what the
claim announced — down to the byte count — is what is collected, even if the
space changes in between. Only a digest of the token is stored: reading this
table yields nothing anyone could present.
*/

// HandoffClaimTTL is how long a claim is good for. The standard's ceiling is
// five minutes, and there is no reason to sit under it: the browser opens the
// receiving service in the same second the claim is issued.
const HandoffClaimTTL = 5 * time.Minute

// HandoffFormat is the one format umm hands over. The backup export carries
// ids and canvas coordinates that a document has no use for; the outline is
// the person's sentences in the order the graph puts them, which is what a
// document or a deck should start from.
const HandoffFormat = "markdown"

// ErrHandoffClaimNotFound covers spent, expired and never-issued alike. The
// standard says not to tell those apart, and a stranger holding a token gets
// nothing from the difference.
var ErrHandoffClaimNotFound = errors.New("handoff claim not found")

// HandoffDocument is what a claim hands over.
type HandoffDocument struct {
	SpaceID     uuid.UUID
	IssuedBy    uuid.UUID
	Filename    string
	ContentType string
	Body        []byte
	ExpiresAt   time.Time
}

// IssueHandoffClaim stores the document under a new token and returns the
// token. The token is 256 bits of randomness, URL-safe so it can travel in a
// query string without escaping; the standard asks for at least 128.
func (s *Store) IssueHandoffClaim(ctx context.Context, doc HandoffDocument) (string, time.Time, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", time.Time{}, err
	}
	token := base64.RawURLEncoding.EncodeToString(buf)
	expires := time.Now().Add(HandoffClaimTTL)
	// Expired rows are nobody's business any more. Swept here rather than by a
	// scheduler because issuing is the only moment the table grows.
	_, _ = s.Pool.Exec(ctx, `DELETE FROM handoff_claims WHERE expires_at < now() - interval '1 hour'`)
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO handoff_claims(claim_digest,space_id,issued_by,filename,content_type,body,expires_at)
		VALUES($1,$2,$3,$4,$5,$6,$7)`,
		handoffDigest(token), doc.SpaceID, doc.IssuedBy, doc.Filename, doc.ContentType, doc.Body, expires)
	if err != nil {
		return "", time.Time{}, err
	}
	return token, expires, nil
}

// RedeemHandoffClaim hands over the document for a token and spends the token.
//
// Single use is the delete: two collectors racing for the same claim both run
// the same statement, and only one of them gets a row back. Expiry is checked
// in the same statement, so an expired claim is not collectable in the window
// before the sweep removes it.
func (s *Store) RedeemHandoffClaim(ctx context.Context, token string) (HandoffDocument, error) {
	var doc HandoffDocument
	err := s.Pool.QueryRow(ctx, `
		DELETE FROM handoff_claims
		WHERE claim_digest=$1 AND expires_at > now()
		RETURNING space_id,issued_by,filename,content_type,body,expires_at`, handoffDigest(token)).
		Scan(&doc.SpaceID, &doc.IssuedBy, &doc.Filename, &doc.ContentType, &doc.Body, &doc.ExpiresAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return HandoffDocument{}, ErrHandoffClaimNotFound
		}
		return HandoffDocument{}, err
	}
	return doc, nil
}

func handoffDigest(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}
