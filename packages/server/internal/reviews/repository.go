package reviews

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	repomemory "co-review/server/internal/repos"
)

type Repository struct {
	db       *sql.DB
	failOnce map[string]error
}

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

func (r *Repository) AcceptCommentAsDecision(ctx context.Context, reviewID, commentID string, entry repomemory.MemoryEntry) (Comment, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Comment{}, err
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx, `UPDATE review_comments SET status=? WHERE review_id=? AND id=?`, CommentStatusAcceptedDecision, reviewID, commentID)
	if err != nil {
		return Comment{}, fmt.Errorf("update comment status: %w", err)
	}
	if err := requireAffected(res, sql.ErrNoRows); err != nil {
		return Comment{}, err
	}
	if err := r.fail("create_memory"); err != nil {
		return Comment{}, err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO repo_memory(id,repo_id,type,key,content,dimension,source_mr,expires_at) VALUES(?,?,?,?,?,?,?,?) ON CONFLICT(repo_id,type,key) DO UPDATE SET content=excluded.content, dimension=excluded.dimension, source_mr=excluded.source_mr, expires_at=excluded.expires_at, updated_at=datetime('now')`, entry.ID, entry.RepoID, entry.Type, entry.Key, entry.Content, nullableString(entry.Dimension), nullableString(entry.SourceMR), nullableTime(entry.ExpiresAt))
	if err != nil {
		return Comment{}, fmt.Errorf("create memory: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return Comment{}, err
	}
	return r.GetComment(ctx, reviewID, commentID)
}

func (r *Repository) InsertReview(ctx context.Context, review Review) error {
	if err := r.fail("insert_review"); err != nil {
		return err
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO reviews(id,repo_id,mr_id,mr_url,mr_title,base_sha,head_sha,start_sha,model_used,status,auto_publish,scores,verdict,error) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, review.ID, review.RepoID, review.MRID, review.MRURL, review.MRTitle, review.BaseSHA, review.HeadSHA, review.StartSHA, review.ModelUsed, review.Status, 0, nullableJSON(review.Scores), nullableString(review.Verdict), nullableJSON(review.Error))
	if err != nil {
		return fmt.Errorf("insert review: %w", err)
	}
	return nil
}

func (r *Repository) UpdateReviewState(ctx context.Context, id, status string, scores json.RawMessage, verdict string, reviewErr json.RawMessage, completed bool) error {
	if err := r.fail("update_review_state:" + status); err != nil {
		return err
	}
	if err := r.fail("update_review_state"); err != nil {
		return err
	}
	completedAt := any(nil)
	if completed {
		completedAt = time.Now().UTC().Format(time.RFC3339)
	}
	_, err := r.db.ExecContext(ctx, `UPDATE reviews SET status=?, scores=?, verdict=?, error=?, completed_at=? WHERE id=?`, status, nullableJSON(scores), nullableString(verdict), nullableJSON(reviewErr), completedAt, id)
	if err != nil {
		return fmt.Errorf("update review state: %w", err)
	}
	return nil
}

func (r *Repository) UpdateReviewContext(ctx context.Context, review Review) error {
	if err := r.fail("update_review_context"); err != nil {
		return err
	}
	_, err := r.db.ExecContext(ctx, `UPDATE reviews SET mr_url=?, mr_title=?, base_sha=?, start_sha=?, head_sha=? WHERE id=?`, review.MRURL, review.MRTitle, review.BaseSHA, review.StartSHA, review.HeadSHA, review.ID)
	if err != nil {
		return fmt.Errorf("update review context: %w", err)
	}
	return nil
}

func (r *Repository) InsertComments(ctx context.Context, comments []Comment) error {
	if err := r.fail("insert_comments"); err != nil {
		return err
	}
	for _, c := range comments {
		_, err := r.db.ExecContext(ctx, `INSERT INTO review_comments(id,review_id,dimension,severity,file,line_start,line_end,evidence,why,suggestion_snippet,status) VALUES(?,?,?,?,?,?,?,?,?,?,?)`, c.ID, c.ReviewID, c.Dimension, c.Severity, c.File, c.LineStart, c.LineEnd, c.Evidence, c.Why, c.SuggestionSnippet, c.Status)
		if err != nil {
			return fmt.Errorf("insert review comment: %w", err)
		}
	}
	return nil
}

func (r *Repository) ListReviews(ctx context.Context) ([]Review, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT reviews.id, repo_id, repos.name, repos.url, repos.platform, mr_id, mr_url, mr_title, base_sha, start_sha, head_sha, status, COALESCE(scores,''), COALESCE(verdict,''), model_used, COALESCE(error,''), reviews.created_at, reviews.completed_at FROM reviews JOIN repos ON repos.id=reviews.repo_id ORDER BY reviews.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanReviews(rows)
}

func (r *Repository) GetReview(ctx context.Context, id string) (Review, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT reviews.id, repo_id, repos.name, repos.url, repos.platform, mr_id, mr_url, mr_title, base_sha, start_sha, head_sha, status, COALESCE(scores,''), COALESCE(verdict,''), model_used, COALESCE(error,''), reviews.created_at, reviews.completed_at FROM reviews JOIN repos ON repos.id=reviews.repo_id WHERE reviews.id=?`, id)
	if err != nil {
		return Review{}, err
	}
	defer rows.Close()
	reviews, err := scanReviews(rows)
	if err != nil {
		return Review{}, err
	}
	if len(reviews) == 0 {
		return Review{}, sql.ErrNoRows
	}
	return reviews[0], nil
}

func (r *Repository) ListComments(ctx context.Context, reviewID string) ([]Comment, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, review_id, dimension, severity, file, line_start, line_end, evidence, why, COALESCE(suggestion_snippet,''), status, created_at FROM review_comments WHERE review_id=? ORDER BY created_at, id`, reviewID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanComments(rows)
}

func (r *Repository) GetComment(ctx context.Context, reviewID, commentID string) (Comment, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, review_id, dimension, severity, file, line_start, line_end, evidence, why, COALESCE(suggestion_snippet,''), status, created_at FROM review_comments WHERE review_id=? AND id=?`, reviewID, commentID)
	if err != nil {
		return Comment{}, err
	}
	defer rows.Close()
	comments, err := scanComments(rows)
	if err != nil {
		return Comment{}, err
	}
	if len(comments) == 0 {
		return Comment{}, sql.ErrNoRows
	}
	return comments[0], nil
}

func (r *Repository) UpdateCommentStatus(ctx context.Context, reviewID, commentID, status string) (Comment, error) {
	res, err := r.db.ExecContext(ctx, `UPDATE review_comments SET status=? WHERE review_id=? AND id=?`, status, reviewID, commentID)
	if err != nil {
		return Comment{}, fmt.Errorf("update comment status: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return Comment{}, err
	}
	if affected == 0 {
		return Comment{}, sql.ErrNoRows
	}
	return r.GetComment(ctx, reviewID, commentID)
}

func (r *Repository) fail(op string) error {
	if r.failOnce == nil {
		return nil
	}
	err := r.failOnce[op]
	if err != nil {
		delete(r.failOnce, op)
	}
	return err
}

func scanReviews(rows *sql.Rows) ([]Review, error) {
	var out []Review
	for rows.Next() {
		var rv Review
		var scores, errJSON, completed sql.NullString
		var created string
		if err := rows.Scan(&rv.ID, &rv.RepoID, &rv.ProjectPath, &rv.ProjectURL, &rv.Platform, &rv.MRID, &rv.MRURL, &rv.MRTitle, &rv.BaseSHA, &rv.StartSHA, &rv.HeadSHA, &rv.Status, &scores, &rv.Verdict, &rv.ModelUsed, &errJSON, &created, &completed); err != nil {
			return nil, err
		}
		if scores.Valid && scores.String != "" {
			rv.Scores = json.RawMessage(scores.String)
		}
		if errJSON.Valid && errJSON.String != "" {
			rv.Error = json.RawMessage(errJSON.String)
		}
		rv.CreatedAt = parseDBTime(created)
		if completed.Valid && completed.String != "" {
			t := parseDBTime(completed.String)
			rv.CompletedAt = &t
		}
		out = append(out, rv)
	}
	return out, rows.Err()
}

func scanComments(rows *sql.Rows) ([]Comment, error) {
	var comments []Comment
	for rows.Next() {
		var c Comment
		var lineStart, lineEnd sql.NullInt64
		var created string
		if err := rows.Scan(&c.ID, &c.ReviewID, &c.Dimension, &c.Severity, &c.File, &lineStart, &lineEnd, &c.Evidence, &c.Why, &c.SuggestionSnippet, &c.Status, &created); err != nil {
			return nil, err
		}
		if lineStart.Valid {
			v := int(lineStart.Int64)
			c.LineStart = &v
		}
		if lineEnd.Valid {
			v := int(lineEnd.Int64)
			c.LineEnd = &v
		}
		c.CreatedAt = parseDBTime(created)
		comments = append(comments, c)
	}
	return comments, rows.Err()
}

func nullableJSON(raw json.RawMessage) any {
	if len(raw) == 0 {
		return nil
	}
	return string(raw)
}
func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}
func nullableTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.UTC().Format(time.RFC3339)
}
func requireAffected(res sql.Result, notFound error) error {
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return notFound
	}
	return nil
}
func parseDBTime(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	if t.IsZero() {
		t, _ = time.Parse("2006-01-02 15:04:05", s)
	}
	return t
}
func stableID(prefix, value string) string {
	return prefix + "_" + strconv.FormatUint(fnv64(value), 36)
}
func fnv64(s string) uint64 {
	var h uint64 = 1469598103934665603
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= 1099511628211
	}
	return h
}
