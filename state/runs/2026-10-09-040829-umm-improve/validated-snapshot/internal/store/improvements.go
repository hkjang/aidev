package store

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type SearchOptions struct {
	Query       string
	SpaceID     *uuid.UUID
	Kind        string
	UpdatedFrom *time.Time
	UpdatedTo   *time.Time
	Offset      int
	Limit       int
}

type HybridSearchPage struct {
	Notes      []NoteSearchResult `json:"notes"`
	NextOffset int                `json:"-"`
	HasMore    bool               `json:"-"`
}

type Backlink struct {
	Edge      Edge   `json:"edge"`
	Note      Note   `json:"note"`
	Direction string `json:"direction"`
}

type ReviewItem struct {
	ID        uuid.UUID `json:"id"`
	SpaceID   uuid.UUID `json:"spaceId"`
	SpaceName string    `json:"spaceName"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	Pinned    bool      `json:"pinned"`
	UpdatedAt time.Time `json:"updatedAt"`
	Reason    string    `json:"reason"`
}

type ReviewDream struct {
	ID              uuid.UUID `json:"id"`
	SpaceID         uuid.UUID `json:"spaceId"`
	SpaceName       string    `json:"spaceName"`
	Content         string    `json:"content"`
	Rationale       string    `json:"rationale"`
	SuggestedAction string    `json:"suggestedAction"`
	GeneratedAt     time.Time `json:"generatedAt"`
}

type ReviewActivity struct {
	ID        uuid.UUID `json:"id"`
	NoteID    uuid.UUID `json:"noteId"`
	SpaceID   uuid.UUID `json:"spaceId"`
	SpaceName string    `json:"spaceName"`
	Author    string    `json:"author"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"createdAt"`
}

type OnboardingProgress struct {
	CompletedAt *time.Time       `json:"completedAt,omitempty"`
	Percent     int              `json:"percent"`
	Steps       []OnboardingStep `json:"steps"`
}

type OnboardingStep struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Done        bool   `json:"done"`
	Target      string `json:"target"`
}

type TodayReview struct {
	Review     []ReviewItem       `json:"review"`
	Orphans    []ReviewItem       `json:"orphans"`
	Dreams     []ReviewDream      `json:"dreams"`
	Activity   []ReviewActivity   `json:"activity"`
	Onboarding OnboardingProgress `json:"onboarding"`
	Counts     map[string]int     `json:"counts"`
}

type Comment struct {
	ID         uuid.UUID  `json:"id"`
	NoteID     uuid.UUID  `json:"noteId"`
	AuthorID   uuid.UUID  `json:"authorId"`
	Author     string     `json:"author"`
	Username   string     `json:"username"`
	ParentID   *uuid.UUID `json:"parentId,omitempty"`
	Body       string     `json:"body"`
	Mentions   []string   `json:"mentions"`
	ResolvedAt *time.Time `json:"resolvedAt,omitempty"`
	ResolvedBy *uuid.UUID `json:"resolvedBy,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
	UpdatedAt  time.Time  `json:"updatedAt"`
}

// Hybrid search is intentionally approximate after PostgreSQL has considered
// the full accessible corpus. Keeping the lexical candidate set bounded avoids
// loading every match for a broad query into the API process while still
// allowing an old or deep-body match to compete on relevance.
const hybridLexicalCandidateLimit = 500

var ErrInvalidParentComment = errors.New("invalid parent comment")

func normalized(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func lexicalScore(query, title, content, space string) (float64, []string) {
	q := normalized(query)
	if q == "" {
		return 0, nil
	}
	title, content, space = normalized(title), normalized(content), normalized(space)
	score := 0.0
	reasons := []string{}
	if strings.Contains(title, q) {
		score += .72
		reasons = append(reasons, "제목 일치")
		if title == q {
			score += .18
			reasons = append(reasons, "제목 정확히 일치")
		}
	}
	if strings.Contains(content, q) {
		score += .55
		reasons = append(reasons, "본문 일치")
		if content == q {
			score += .12
			reasons = append(reasons, "본문 정확히 일치")
		}
	}
	if strings.Contains(space, q) {
		score += .25
		reasons = append(reasons, "공간 이름 일치")
	}
	terms := strings.Fields(q)
	matched := 0
	for _, term := range terms {
		if strings.Contains(title+" "+content+" "+space, term) {
			matched++
		}
	}
	if len(terms) > 0 {
		score += .35 * float64(matched) / float64(len(terms))
	}
	return math.Min(score, 1), reasons
}

func (s *Store) noteViewAccess(ctx context.Context, userID, noteID uuid.UUID) (bool, error) {
	var allowed bool
	err := s.Pool.QueryRow(ctx, `
		SELECT EXISTS(
		  SELECT 1 FROM notes n
		  JOIN spaces sp ON sp.id=n.space_id
		  LEFT JOIN space_members sm ON sm.space_id=sp.id AND sm.user_id=$2
		  WHERE n.id=$1 AND n.deleted_at IS NULL AND (sp.owner_id=$2 OR sm.user_id=$2))`, noteID, userID).Scan(&allowed)
	return allowed, err
}
func (s *Store) Backlinks(ctx context.Context, userID, noteID uuid.UUID) ([]Backlink, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT e.id,e.space_id,e.source_note_id,e.target_note_id,e.relation,e.origin,e.confidence,e.reason,
		       n.id,n.space_id,n.author_id,n.content,n.title,n.color,n.kind,n.source,n.ai_excluded,
		       n.x,n.y,n.width,n.height,n.rotation,n.version,n.created_at,n.updated_at,
		       CASE WHEN e.target_note_id=$1 THEN 'incoming' ELSE 'outgoing' END
		FROM notes anchor
		JOIN spaces sp ON sp.id=anchor.space_id
		LEFT JOIN space_members sm ON sm.space_id=sp.id AND sm.user_id=$2
		JOIN note_edges e ON e.space_id=anchor.space_id
		JOIN notes n ON n.id=CASE WHEN e.target_note_id=anchor.id THEN e.source_note_id ELSE e.target_note_id END
		  AND n.space_id=anchor.space_id
		WHERE anchor.id=$1 AND anchor.deleted_at IS NULL AND (sp.owner_id=$2 OR sm.user_id=$2)
		  AND (e.source_note_id=anchor.id OR e.target_note_id=anchor.id) AND e.space_id=anchor.space_id
		  AND n.deleted_at IS NULL
		ORDER BY n.updated_at DESC`, noteID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Backlink{}
	for rows.Next() {
		var item Backlink
		if err := rows.Scan(&item.Edge.ID, &item.Edge.SpaceID, &item.Edge.SourceID, &item.Edge.TargetID, &item.Edge.Relation,
			&item.Edge.Origin, &item.Edge.Confidence, &item.Edge.Reason,
			&item.Note.ID, &item.Note.SpaceID, &item.Note.AuthorID, &item.Note.Content, &item.Note.Title, &item.Note.Color,
			&item.Note.Kind, &item.Note.Source, &item.Note.AIExcluded, &item.Note.X, &item.Note.Y, &item.Note.Width,
			&item.Note.Height, &item.Note.Rotation, &item.Note.Version, &item.Note.CreatedAt, &item.Note.UpdatedAt, &item.Direction); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	if len(out) == 0 {
		allowed, accessErr := s.noteViewAccess(ctx, userID, noteID)
		if accessErr != nil {
			return nil, accessErr
		}
		if !allowed {
			return nil, pgx.ErrNoRows
		}
	}
	return out, nil
}

func (s *Store) NoteByID(ctx context.Context, userID, noteID uuid.UUID) (Note, error) {
	n, _, err := s.NoteByIDWithEditAccess(ctx, userID, noteID)
	return n, err
}

func (s *Store) NoteByIDWithEditAccess(ctx context.Context, userID, noteID uuid.UUID) (Note, bool, error) {
	var n Note
	var canEdit bool
	err := s.Pool.QueryRow(ctx, `
		SELECT n.id,n.space_id,n.author_id,n.content,n.title,n.color,n.kind,n.source,n.ai_excluded,
		       n.x,n.y,n.width,n.height,n.rotation,n.version,n.created_at,n.updated_at,
		       (sp.owner_id=$2 OR sm.permission IN ('edit','manage'))
		FROM notes n
		JOIN spaces sp ON sp.id=n.space_id
		LEFT JOIN space_members sm ON sm.space_id=sp.id AND sm.user_id=$2
		WHERE n.id=$1 AND n.deleted_at IS NULL AND (sp.owner_id=$2 OR sm.user_id=$2)`, noteID, userID).
		Scan(&n.ID, &n.SpaceID, &n.AuthorID, &n.Content, &n.Title, &n.Color, &n.Kind, &n.Source, &n.AIExcluded,
			&n.X, &n.Y, &n.Width, &n.Height, &n.Rotation, &n.Version, &n.CreatedAt, &n.UpdatedAt, &canEdit)
	return n, canEdit, err
}

func (s *Store) TodayReview(ctx context.Context, userID uuid.UUID) (TodayReview, error) {
	out := TodayReview{Review: []ReviewItem{}, Orphans: []ReviewItem{}, Dreams: []ReviewDream{}, Activity: []ReviewActivity{}, Counts: map[string]int{}}
	reviewRows, err := s.Pool.Query(ctx, `
		SELECT n.id,n.space_id,sp.name,n.title,left(n.content,500),COALESCE(nr.pinned,false),n.updated_at,
		       CASE WHEN nr.pinned THEN '중요 고정' WHEN nr.review_at IS NOT NULL THEN '다시 볼 시간' ELSE '오랫동안 열지 않음' END
		FROM notes n JOIN spaces sp ON sp.id=n.space_id
		LEFT JOIN note_reviews nr ON nr.note_id=n.id AND nr.user_id=$1
		WHERE n.deleted_at IS NULL AND n.archived=false
		  AND (sp.owner_id=$1 OR EXISTS(SELECT 1 FROM space_members sm WHERE sm.space_id=sp.id AND sm.user_id=$1))
		  AND (COALESCE(nr.pinned,false) OR (nr.review_at IS NOT NULL AND nr.review_at<=now()) OR
		       (nr.review_at IS NULL AND COALESCE(nr.reviewed_at,n.updated_at)<now()-interval '14 days'))
		ORDER BY COALESCE(nr.pinned,false) DESC,COALESCE(nr.review_at,nr.reviewed_at,n.updated_at),n.id LIMIT 8`, userID)
	if err != nil {
		return out, err
	}
	for reviewRows.Next() {
		var item ReviewItem
		if err := reviewRows.Scan(&item.ID, &item.SpaceID, &item.SpaceName, &item.Title, &item.Content, &item.Pinned, &item.UpdatedAt, &item.Reason); err != nil {
			reviewRows.Close()
			return out, err
		}
		out.Review = append(out.Review, item)
	}
	if err := reviewRows.Err(); err != nil {
		reviewRows.Close()
		return out, err
	}
	reviewRows.Close()

	orphanRows, err := s.Pool.Query(ctx, `
		SELECT n.id,n.space_id,sp.name,n.title,left(n.content,500),COALESCE(nr.pinned,false),n.updated_at,'연결되지 않은 생각'
		FROM notes n JOIN spaces sp ON sp.id=n.space_id
		LEFT JOIN note_reviews nr ON nr.note_id=n.id AND nr.user_id=$1
		WHERE n.deleted_at IS NULL AND n.archived=false
		  AND (sp.owner_id=$1 OR EXISTS(SELECT 1 FROM space_members sm WHERE sm.space_id=sp.id AND sm.user_id=$1))
		  AND NOT EXISTS(
		    SELECT 1 FROM note_edges e
		    JOIN notes linked ON linked.id=CASE WHEN e.source_note_id=n.id THEN e.target_note_id ELSE e.source_note_id END
		    WHERE (e.source_note_id=n.id OR e.target_note_id=n.id) AND linked.deleted_at IS NULL
		  )
		ORDER BY n.updated_at DESC,n.id DESC LIMIT 6`, userID)
	if err != nil {
		return out, err
	}
	for orphanRows.Next() {
		var item ReviewItem
		if err := orphanRows.Scan(&item.ID, &item.SpaceID, &item.SpaceName, &item.Title, &item.Content, &item.Pinned, &item.UpdatedAt, &item.Reason); err != nil {
			orphanRows.Close()
			return out, err
		}
		out.Orphans = append(out.Orphans, item)
	}
	if err := orphanRows.Err(); err != nil {
		orphanRows.Close()
		return out, err
	}
	orphanRows.Close()

	dreamRows, err := s.Pool.Query(ctx, `
		SELECT d.dream_id,d.space_id,sp.name,d.content,d.rationale,d.suggested_action,d.generated_at
		FROM dream_notes d JOIN spaces sp ON sp.id=d.space_id
		WHERE d.user_id=$1 AND d.status IN ('created','exposed')
		  AND (sp.owner_id=$1 OR EXISTS(SELECT 1 FROM space_members sm WHERE sm.space_id=sp.id AND sm.user_id=$1))
		ORDER BY d.generated_at DESC,d.dream_id DESC LIMIT 5`, userID)
	if err != nil {
		return out, err
	}
	for dreamRows.Next() {
		var item ReviewDream
		if err := dreamRows.Scan(&item.ID, &item.SpaceID, &item.SpaceName, &item.Content, &item.Rationale, &item.SuggestedAction, &item.GeneratedAt); err != nil {
			dreamRows.Close()
			return out, err
		}
		out.Dreams = append(out.Dreams, item)
	}
	if err := dreamRows.Err(); err != nil {
		dreamRows.Close()
		return out, err
	}
	dreamRows.Close()

	activityRows, err := s.Pool.Query(ctx, `
		SELECT c.id,c.note_id,n.space_id,sp.name,u.display_name,left(c.body,500),c.created_at
		FROM note_comments c JOIN notes n ON n.id=c.note_id JOIN spaces sp ON sp.id=n.space_id JOIN users u ON u.id=c.author_id
		WHERE c.deleted_at IS NULL AND n.deleted_at IS NULL AND c.author_id<>$1
		  AND (sp.owner_id=$1 OR EXISTS(SELECT 1 FROM space_members sm WHERE sm.space_id=sp.id AND sm.user_id=$1))
		  AND COALESCE((SELECT review_digest FROM user_preferences WHERE user_id=$1),true)
		ORDER BY c.created_at DESC,c.id DESC LIMIT 8`, userID)
	if err != nil {
		return out, err
	}
	for activityRows.Next() {
		var item ReviewActivity
		if err := activityRows.Scan(&item.ID, &item.NoteID, &item.SpaceID, &item.SpaceName, &item.Author, &item.Body, &item.CreatedAt); err != nil {
			activityRows.Close()
			return out, err
		}
		out.Activity = append(out.Activity, item)
	}
	if err := activityRows.Err(); err != nil {
		activityRows.Close()
		return out, err
	}
	activityRows.Close()

	var completedAt *time.Time
	var spaces, notes, edges, comments, dreamActions int
	err = s.Pool.QueryRow(ctx, `
		SELECT p.onboarding_completed_at,
		 (SELECT count(*) FROM spaces sp WHERE sp.owner_id=$1),
		 (SELECT count(*) FROM notes n WHERE n.author_id=$1 AND n.deleted_at IS NULL),
		 (SELECT count(*) FROM note_edges e WHERE e.created_by=$1),
		 (SELECT count(*) FROM note_comments c WHERE c.author_id=$1 AND c.deleted_at IS NULL),
		 (SELECT count(*) FROM dream_feedback f WHERE f.user_id=$1)
		FROM user_preferences p WHERE p.user_id=$1`, userID).
		Scan(&completedAt, &spaces, &notes, &edges, &comments, &dreamActions)
	if err != nil {
		return out, err
	}
	steps := []OnboardingStep{
		{Key: "space", Label: "생각 공간 확인", Description: "내 공간을 열어 구조를 살펴보세요.", Done: spaces > 0, Target: "/"},
		{Key: "note", Label: "첫 생각 붙이기", Description: "짧은 문장 하나면 충분합니다.", Done: notes > 0, Target: "/"},
		{Key: "connect", Label: "생각 연결하기", Description: "관련 있는 두 생각을 선으로 연결하세요.", Done: edges > 0, Target: "/"},
		{Key: "collaborate", Label: "대화 또는 Dream 반응", Description: "댓글을 남기거나 Dream에 반응하세요.", Done: comments > 0 || dreamActions > 0, Target: "/dreams"},
	}
	done := 0
	for _, step := range steps {
		if step.Done {
			done++
		}
	}
	// Doing all four things the guide asks for is finishing the guide.
	//
	// It used to end only when someone found the quiet grey "안내 마치기" link,
	// so a person who had done every step still met the card every morning: it
	// was measured sitting at 100% and occupying 517px above every piece of real
	// work on the page. A checklist with nothing left to check is not guidance,
	// it is furniture.
	if completedAt == nil && done == len(steps) {
		finished, finishErr := s.finishOnboarding(ctx, userID)
		if finishErr != nil {
			// Not worth failing the whole page over; the card simply stays one
			// more load, which is what it did before this existed.
			slog.Warn("could not close the finished onboarding guide", "user_id", userID, "error", finishErr)
		} else {
			completedAt = finished
		}
	}
	out.Onboarding = OnboardingProgress{CompletedAt: completedAt, Percent: done * 100 / len(steps), Steps: steps}

	// The tiles are counts, so they have to count.
	//
	// They used to report len() of lists the queries capped at five to eight, so
	// a workspace with two thousand unconnected thoughts said six. That is not a
	// sample size on the page — it is a number beside the words "waiting to be
	// connected", and reading it as the total is the only way to read it. The
	// lists below stay capped; they are a preview and look like one.
	if err := s.countTodayTotals(ctx, userID, &out); err != nil {
		return out, err
	}
	return out, nil
}

// countTodayTotals fills the summary numbers with totals rather than the length
// of the previewed lists. One round trip, because four counts on the landing
// page should not be four round trips.
func (s *Store) countTodayTotals(ctx context.Context, userID uuid.UUID, out *TodayReview) error {
	// Each count repeats its list's WHERE clause exactly. A count that filters
	// differently from the list beneath it is worse than the cap it replaces: the
	// number and the rows would disagree and neither would be wrong-looking.
	var review, orphans, dreams, activity int
	err := s.Pool.QueryRow(ctx, `
		SELECT
		 (SELECT count(*) FROM notes n JOIN spaces sp ON sp.id=n.space_id
		    LEFT JOIN note_reviews nr ON nr.note_id=n.id AND nr.user_id=$1
		    WHERE n.deleted_at IS NULL AND n.archived=false
		      AND (sp.owner_id=$1 OR EXISTS(SELECT 1 FROM space_members sm WHERE sm.space_id=sp.id AND sm.user_id=$1))
		      AND (COALESCE(nr.pinned,false) OR (nr.review_at IS NOT NULL AND nr.review_at<=now()) OR
		           (nr.review_at IS NULL AND COALESCE(nr.reviewed_at,n.updated_at)<now()-interval '14 days'))),
		 (SELECT count(*) FROM notes n JOIN spaces sp ON sp.id=n.space_id
		    WHERE n.deleted_at IS NULL AND n.archived=false
		      AND (sp.owner_id=$1 OR EXISTS(SELECT 1 FROM space_members sm WHERE sm.space_id=sp.id AND sm.user_id=$1))
		      AND NOT EXISTS(
		        SELECT 1 FROM note_edges e
		        JOIN notes linked ON linked.id=CASE WHEN e.source_note_id=n.id THEN e.target_note_id ELSE e.source_note_id END
		        WHERE (e.source_note_id=n.id OR e.target_note_id=n.id) AND linked.deleted_at IS NULL)),
		 (SELECT count(*) FROM dream_notes d JOIN spaces sp ON sp.id=d.space_id
		    WHERE d.user_id=$1 AND d.status IN ('created','exposed')
		      AND (sp.owner_id=$1 OR EXISTS(SELECT 1 FROM space_members sm WHERE sm.space_id=sp.id AND sm.user_id=$1))),
		 (SELECT count(*) FROM note_comments c JOIN notes n ON n.id=c.note_id JOIN spaces sp ON sp.id=n.space_id
		    WHERE c.deleted_at IS NULL AND n.deleted_at IS NULL AND c.author_id<>$1
		      AND (sp.owner_id=$1 OR EXISTS(SELECT 1 FROM space_members sm WHERE sm.space_id=sp.id AND sm.user_id=$1))
		      AND COALESCE((SELECT review_digest FROM user_preferences WHERE user_id=$1),true))
		`, userID).Scan(&review, &orphans, &dreams, &activity)
	if err != nil {
		return err
	}
	out.Counts["review"] = review
	out.Counts["orphans"] = orphans
	out.Counts["dreams"] = dreams
	out.Counts["activity"] = activity
	return nil
}

func (s *Store) UpdateReview(ctx context.Context, userID, noteID uuid.UUID, snoozeDays *int, pinned *bool, complete bool) (ReviewItem, error) {
	var item ReviewItem
	err := s.Pool.QueryRow(ctx, `
		WITH allowed AS (
		  SELECT n.id,n.space_id,sp.name,n.title,left(n.content,500) AS content,n.updated_at
		  FROM notes n JOIN spaces sp ON sp.id=n.space_id
		  LEFT JOIN space_members sm ON sm.space_id=sp.id AND sm.user_id=$2
		  WHERE n.id=$1 AND n.deleted_at IS NULL AND (sp.owner_id=$2 OR sm.user_id=$2)
		), changed AS (
		  INSERT INTO note_reviews(user_id,note_id,reviewed_at,review_at,pinned,updated_at)
		  SELECT $2,$1,CASE WHEN $5 THEN now() END,
		    CASE WHEN $5 AND $3::integer IS NOT NULL THEN now()+make_interval(days=>$3) END,
		    COALESCE($4,false),now() FROM allowed
		  ON CONFLICT(user_id,note_id) DO UPDATE SET
		    reviewed_at=CASE WHEN $5 THEN now() ELSE note_reviews.reviewed_at END,
		    review_at=CASE WHEN NOT $5 THEN note_reviews.review_at WHEN $3::integer IS NULL THEN NULL ELSE now()+make_interval(days=>$3) END,
		    pinned=COALESCE($4,note_reviews.pinned),updated_at=now()
		  RETURNING note_id,pinned
		)
		SELECT a.id,a.space_id,a.name,a.title,a.content,c.pinned,a.updated_at,
		  CASE WHEN $5 THEN '검토 완료' WHEN c.pinned THEN '중요 고정' ELSE '고정 해제' END
		FROM allowed a JOIN changed c ON c.note_id=a.id`,
		noteID, userID, snoozeDays, pinned, complete).
		Scan(&item.ID, &item.SpaceID, &item.SpaceName, &item.Title, &item.Content, &item.Pinned, &item.UpdatedAt, &item.Reason)
	return item, err
}

func (s *Store) CompleteOnboarding(ctx context.Context, userID uuid.UUID) error {
	_, err := s.finishOnboarding(ctx, userID)
	return err
}

// finishOnboarding marks the guide done and reports when it was, so a caller
// that has just finished it can say so without inventing its own clock and
// without a second round trip. COALESCE keeps the original moment: finishing
// twice does not move the date.
func (s *Store) finishOnboarding(ctx context.Context, userID uuid.UUID) (*time.Time, error) {
	var at time.Time
	err := s.Pool.QueryRow(ctx, `UPDATE user_preferences SET onboarding_completed_at=COALESCE(onboarding_completed_at,now()),updated_at=now()
		WHERE user_id=$1 RETURNING onboarding_completed_at`, userID).Scan(&at)
	if err != nil {
		return nil, err
	}
	return &at, nil
}

func scanComment(row pgx.Row) (Comment, error) {
	var c Comment
	err := row.Scan(&c.ID, &c.NoteID, &c.AuthorID, &c.Author, &c.Username, &c.ParentID, &c.Body, &c.Mentions,
		&c.ResolvedAt, &c.ResolvedBy, &c.CreatedAt, &c.UpdatedAt)
	if c.Mentions == nil {
		c.Mentions = []string{}
	}
	return c, err
}

const commentSelect = `SELECT c.id,c.note_id,c.author_id,u.display_name,u.username::text,c.parent_id,c.body,
	COALESCE((SELECT array_agg(mu.username::text ORDER BY mu.username::text) FROM comment_mentions cm JOIN users mu ON mu.id=cm.user_id WHERE cm.comment_id=c.id),'{}'::text[]),
	c.resolved_at,c.resolved_by,c.created_at,c.updated_at`

const mentionTrailingSyntax = ".,;:!?…，。！？；：)]}>\"'”’"

func (s *Store) ListComments(ctx context.Context, userID, noteID uuid.UUID) ([]Comment, error) {
	rows, err := s.Pool.Query(ctx, commentSelect+`
		FROM note_comments c
		JOIN users u ON u.id=c.author_id
		JOIN notes n ON n.id=c.note_id AND n.deleted_at IS NULL
		JOIN spaces sp ON sp.id=n.space_id
		LEFT JOIN space_members sm ON sm.space_id=sp.id AND sm.user_id=$2
		WHERE c.note_id=$1 AND c.deleted_at IS NULL AND (sp.owner_id=$2 OR sm.user_id=$2)
		ORDER BY c.created_at,c.id`, noteID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Comment{}
	for rows.Next() {
		c, err := scanComment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	if len(out) == 0 {
		allowed, accessErr := s.noteViewAccess(ctx, userID, noteID)
		if accessErr != nil {
			return nil, accessErr
		}
		if !allowed {
			return nil, pgx.ErrNoRows
		}
	}
	return out, nil
}

func (s *Store) CreateComment(ctx context.Context, userID, noteID uuid.UUID, parentID *uuid.UUID, body string, mentionTokens []string) (Comment, uuid.UUID, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return Comment{}, uuid.Nil, err
	}
	defer tx.Rollback(ctx)
	var spaceID, noteAuthor, spaceOwner uuid.UUID
	var noteAuthorCanView bool
	err = tx.QueryRow(ctx, `
		SELECT n.space_id,n.author_id,sp.owner_id
		FROM notes n JOIN spaces sp ON sp.id=n.space_id
		WHERE n.id=$1 AND n.deleted_at IS NULL
		FOR SHARE OF n,sp`, noteID).
		Scan(&spaceID, &noteAuthor, &spaceOwner)
	if err != nil {
		return Comment{}, uuid.Nil, err
	}
	if spaceOwner != userID {
		var currentMember uuid.UUID
		err = tx.QueryRow(ctx, `
			SELECT user_id FROM space_members
			WHERE space_id=$1 AND user_id=$2
			FOR KEY SHARE`, spaceID, userID).Scan(&currentMember)
		if err != nil {
			return Comment{}, uuid.Nil, err
		}
	}
	if noteAuthor == spaceOwner {
		noteAuthorCanView = true
	} else if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM space_members WHERE space_id=$1 AND user_id=$2)`, spaceID, noteAuthor).Scan(&noteAuthorCanView); err != nil {
		return Comment{}, uuid.Nil, err
	}
	if parentID != nil {
		var valid bool
		if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM note_comments WHERE id=$1 AND note_id=$2 AND deleted_at IS NULL)`, parentID, noteID).Scan(&valid); err != nil {
			return Comment{}, uuid.Nil, err
		}
		if !valid {
			return Comment{}, uuid.Nil, ErrInvalidParentComment
		}
	}
	var commentID uuid.UUID
	if err = tx.QueryRow(ctx, `INSERT INTO note_comments(note_id,author_id,parent_id,body) VALUES($1,$2,$3,$4) RETURNING id`, noteID, userID, parentID, body).Scan(&commentID); err != nil {
		return Comment{}, uuid.Nil, err
	}
	clean := make([]string, 0, len(mentionTokens))
	for _, name := range mentionTokens {
		name = strings.ToLower(strings.TrimSpace(name))
		if name != "" {
			clean = append(clean, name)
		}
	}
	mentioned := map[uuid.UUID]bool{}
	if len(clean) > 0 {
		rows, queryErr := tx.Query(ctx, `
			WITH mention_tokens AS (
			  SELECT lower(token) AS token,ordinality
			  FROM unnest($1::text[]) WITH ORDINALITY AS raw(token,ordinality)
			), resolved AS (
			  SELECT DISTINCT ON (mt.ordinality) u.id,mt.ordinality
			  FROM mention_tokens mt JOIN users u ON
			    char_length(lower(u.username::text))>0 AND
			    left(mt.token,char_length(lower(u.username::text)))=lower(u.username::text) AND
			    translate(substr(mt.token,char_length(lower(u.username::text))+1), $3, '')=''
			  WHERE u.active AND EXISTS(
			    SELECT 1 FROM spaces sp LEFT JOIN space_members sm ON sm.space_id=sp.id AND sm.user_id=u.id
			    WHERE sp.id=$2 AND (sp.owner_id=u.id OR sm.user_id=u.id))
			  ORDER BY mt.ordinality,char_length(lower(u.username::text)) DESC,u.id
			)
			SELECT id FROM resolved ORDER BY ordinality`, clean, spaceID, mentionTrailingSyntax)
		if queryErr != nil {
			return Comment{}, uuid.Nil, queryErr
		}
		targets := []uuid.UUID{}
		for rows.Next() {
			var target uuid.UUID
			if err = rows.Scan(&target); err != nil {
				rows.Close()
				return Comment{}, uuid.Nil, err
			}
			if target != userID && !mentioned[target] {
				mentioned[target] = true
				targets = append(targets, target)
			}
		}
		if err = rows.Err(); err != nil {
			rows.Close()
			return Comment{}, uuid.Nil, err
		}
		rows.Close()
		for _, target := range targets {
			if _, err = tx.Exec(ctx, `INSERT INTO comment_mentions(comment_id,user_id) VALUES($1,$2) ON CONFLICT DO NOTHING`, commentID, target); err != nil {
				return Comment{}, uuid.Nil, err
			}
			if _, err = tx.Exec(ctx, `INSERT INTO notifications(user_id,kind,title,body,resource_type,resource_id,resource_space_id,metadata) VALUES($1,'mention','댓글에서 회원님을 언급했습니다',left($2,500),'note',$3,$4,jsonb_build_object('commentId',$5::uuid))`, target, body, noteID, spaceID, commentID); err != nil {
				return Comment{}, uuid.Nil, err
			}
		}
	}
	if noteAuthor != userID && noteAuthorCanView && !mentioned[noteAuthor] {
		if _, err = tx.Exec(ctx, `INSERT INTO notifications(user_id,kind,title,body,resource_type,resource_id,resource_space_id,metadata) VALUES($1,'comment','내 생각에 새 댓글이 달렸습니다',left($2,500),'note',$3,$4,jsonb_build_object('commentId',$5::uuid))`, noteAuthor, body, noteID, spaceID, commentID); err != nil {
			return Comment{}, uuid.Nil, err
		}
	}
	if _, err = tx.Exec(ctx, `INSERT INTO product_events(user_id,event_name,resource_type,resource_id,metadata) VALUES($1,'comment.created','note',$2,jsonb_build_object('spaceId',$3::uuid))`, userID, noteID, spaceID); err != nil {
		return Comment{}, uuid.Nil, err
	}
	comment, err := scanComment(tx.QueryRow(ctx, commentSelect+` FROM note_comments c JOIN users u ON u.id=c.author_id WHERE c.id=$1`, commentID))
	if err != nil {
		return Comment{}, uuid.Nil, err
	}
	if err = s.AppendSpaceEvent(ctx, tx, userID, spaceID, "comment.created", comment.ID, comment); err != nil {
		return Comment{}, uuid.Nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Comment{}, uuid.Nil, err
	}
	return comment, spaceID, nil
}

func (s *Store) ResolveComment(ctx context.Context, userID, commentID uuid.UUID, resolved bool) (Comment, uuid.UUID, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return Comment{}, uuid.Nil, err
	}
	defer tx.Rollback(ctx)
	var noteID, spaceID, spaceOwner uuid.UUID
	err = tx.QueryRow(ctx, `
		SELECT c.note_id,n.space_id,sp.owner_id
		FROM note_comments c
		JOIN notes n ON n.id=c.note_id
		JOIN spaces sp ON sp.id=n.space_id
		WHERE c.id=$1 AND c.deleted_at IS NULL AND n.deleted_at IS NULL
		FOR UPDATE OF c
		FOR SHARE OF n,sp`, commentID).Scan(&noteID, &spaceID, &spaceOwner)
	if err != nil {
		return Comment{}, uuid.Nil, err
	}
	if spaceOwner != userID {
		var permission string
		if err = tx.QueryRow(ctx, `
			SELECT permission FROM space_members
			WHERE space_id=$1 AND user_id=$2 AND permission IN ('edit','manage')
			FOR SHARE`, spaceID, userID).Scan(&permission); err != nil {
			return Comment{}, uuid.Nil, err
		}
	}
	if _, err = tx.Exec(ctx, `
		UPDATE note_comments
		SET resolved_at=CASE WHEN $3 THEN now() ELSE NULL END,
		    resolved_by=CASE WHEN $3 THEN $2::uuid ELSE NULL END,
		    updated_at=now()
		WHERE id=$1`, commentID, userID, resolved); err != nil {
		return Comment{}, uuid.Nil, err
	}
	comment, err := scanComment(tx.QueryRow(ctx, commentSelect+` FROM note_comments c JOIN users u ON u.id=c.author_id WHERE c.id=$1`, commentID))
	if err != nil {
		return Comment{}, uuid.Nil, err
	}
	if err = s.AppendSpaceEvent(ctx, tx, userID, spaceID, "comment.resolved", comment.ID, comment); err != nil {
		return Comment{}, uuid.Nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Comment{}, uuid.Nil, err
	}
	return comment, spaceID, nil
}

func (s *Store) DeleteComment(ctx context.Context, userID, commentID uuid.UUID) (uuid.UUID, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	defer tx.Rollback(ctx)
	var spaceID, spaceOwner, commentAuthor uuid.UUID
	err = tx.QueryRow(ctx, `
		SELECT n.space_id,sp.owner_id,c.author_id
		FROM note_comments c
		JOIN notes n ON n.id=c.note_id
		JOIN spaces sp ON sp.id=n.space_id
		WHERE c.id=$1 AND c.deleted_at IS NULL AND n.deleted_at IS NULL
		FOR UPDATE OF c
		FOR SHARE OF n,sp`, commentID).Scan(&spaceID, &spaceOwner, &commentAuthor)
	if err != nil {
		return uuid.Nil, err
	}
	if spaceOwner != userID {
		var permission string
		if err = tx.QueryRow(ctx, `
			SELECT permission FROM space_members
			WHERE space_id=$1 AND user_id=$2
			FOR SHARE`, spaceID, userID).Scan(&permission); err != nil {
			return uuid.Nil, err
		}
		if commentAuthor != userID && permission != "manage" {
			return uuid.Nil, pgx.ErrNoRows
		}
	}
	if _, err = tx.Exec(ctx, `UPDATE note_comments SET deleted_at=now(),updated_at=now() WHERE id=$1`, commentID); err != nil {
		return uuid.Nil, err
	}
	if err = s.AppendSpaceEvent(ctx, tx, userID, spaceID, "comment.deleted", commentID, map[string]any{}); err != nil {
		return uuid.Nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return uuid.Nil, err
	}
	return spaceID, nil
}

func (s *Store) Track(ctx context.Context, userID *uuid.UUID, event, resourceType string, resourceID *uuid.UUID, metadata any) {
	raw, _ := json.Marshal(metadata)
	_, _ = s.Pool.Exec(ctx, `INSERT INTO product_events(user_id,event_name,resource_type,resource_id,metadata) VALUES($1,$2,$3,$4,$5)`, userID, event, resourceType, resourceID, raw)
}
