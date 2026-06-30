package repos

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"
)

type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

func (r *Repository) CreateRepo(ctx context.Context, repo Repo) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO repos(id,name,url,platform,default_branch,auto_publish,publish_mode,active) VALUES(?,?,?,?,?,?,?,?)`, repo.ID, repo.Name, repo.URL, repo.Platform, repo.DefaultBranch, boolInt(repo.AutoPublish), repo.PublishMode, boolInt(repo.Active))
	if err != nil {
		return fmt.Errorf("create repo: %w", err)
	}
	return nil
}

func (r *Repository) UpsertRepo(ctx context.Context, repo Repo) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO repos(id,name,url,platform,default_branch,auto_publish,publish_mode,active) VALUES(?,?,?,?,?,?,?,?) ON CONFLICT(name) DO UPDATE SET url=excluded.url, platform=excluded.platform, default_branch=excluded.default_branch, auto_publish=excluded.auto_publish, publish_mode=excluded.publish_mode, active=excluded.active`, repo.ID, repo.Name, repo.URL, repo.Platform, repo.DefaultBranch, boolInt(repo.AutoPublish), repo.PublishMode, boolInt(repo.Active))
	if err != nil {
		return fmt.Errorf("upsert repo: %w", err)
	}
	return nil
}

func (r *Repository) ListRepos(ctx context.Context) ([]Repo, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,name,url,platform,default_branch,auto_publish,publish_mode,active,created_at FROM repos ORDER BY created_at DESC, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRepos(rows)
}

func (r *Repository) GetRepo(ctx context.Context, id string) (Repo, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,name,url,platform,default_branch,auto_publish,publish_mode,active,created_at FROM repos WHERE id=?`, id)
	if err != nil {
		return Repo{}, err
	}
	defer rows.Close()
	repos, err := scanRepos(rows)
	if err != nil {
		return Repo{}, err
	}
	if len(repos) == 0 {
		return Repo{}, sql.ErrNoRows
	}
	return repos[0], nil
}

func (r *Repository) UpdateRepo(ctx context.Context, repo Repo) error {
	res, err := r.db.ExecContext(ctx, `UPDATE repos SET name=?, url=?, platform=?, default_branch=?, auto_publish=?, publish_mode=?, active=? WHERE id=?`, repo.Name, repo.URL, repo.Platform, repo.DefaultBranch, boolInt(repo.AutoPublish), repo.PublishMode, boolInt(repo.Active), repo.ID)
	if err != nil {
		return fmt.Errorf("update repo: %w", err)
	}
	return requireAffected(res, sql.ErrNoRows)
}

func (r *Repository) DeleteRepo(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM repos WHERE id=?`, id)
	if err != nil {
		return fmt.Errorf("delete repo: %w", err)
	}
	return requireAffected(res, sql.ErrNoRows)
}

func (r *Repository) PutModel(ctx context.Context, cfg ModelConfig) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `UPDATE model_configs SET is_active=0 WHERE repo_id=?`, cfg.RepoID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO model_configs(id,repo_id,provider,model_name,api_key_env,is_active) VALUES(?,?,?,?,?,1)`, cfg.ID, cfg.RepoID, cfg.Provider, cfg.ModelName, nullable(cfg.APIKeyEnv)); err != nil {
		return fmt.Errorf("put model config: %w", err)
	}
	return tx.Commit()
}

func (r *Repository) GetActiveModel(ctx context.Context, repoID string) (ModelConfig, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id,repo_id,provider,model_name,COALESCE(api_key_env,''),is_active,created_at FROM model_configs WHERE repo_id=? AND is_active=1 ORDER BY created_at DESC LIMIT 1`, repoID)
	var cfg ModelConfig
	var active int
	var created string
	if err := row.Scan(&cfg.ID, &cfg.RepoID, &cfg.Provider, &cfg.ModelName, &cfg.APIKeyEnv, &active, &created); err != nil {
		return ModelConfig{}, err
	}
	cfg.IsActive = active == 1
	cfg.CreatedAt = parseDBTime(created)
	return cfg, nil
}

func (r *Repository) CreateMemory(ctx context.Context, entry MemoryEntry) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO repo_memory(id,repo_id,type,key,content,dimension,source_mr,expires_at) VALUES(?,?,?,?,?,?,?,?) ON CONFLICT(repo_id,type,key) DO UPDATE SET content=excluded.content, dimension=excluded.dimension, source_mr=excluded.source_mr, expires_at=excluded.expires_at, updated_at=datetime('now')`, entry.ID, entry.RepoID, entry.Type, entry.Key, entry.Content, nullable(entry.Dimension), nullable(entry.SourceMR), nullableTime(entry.ExpiresAt))
	if err != nil {
		return fmt.Errorf("create memory: %w", err)
	}
	return nil
}

func (r *Repository) ListMemory(ctx context.Context, repoID string) ([]MemoryEntry, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,repo_id,type,key,content,COALESCE(dimension,''),COALESCE(source_mr,''),created_at,updated_at,expires_at FROM repo_memory WHERE repo_id=? AND (expires_at IS NULL OR expires_at > datetime('now')) ORDER BY type, key, created_at`, repoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMemory(rows)
}

func (r *Repository) DeleteMemory(ctx context.Context, repoID, memoryID string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM repo_memory WHERE repo_id=? AND id=?`, repoID, memoryID)
	if err != nil {
		return fmt.Errorf("delete memory: %w", err)
	}
	return requireAffected(res, sql.ErrNoRows)
}

func scanRepos(rows *sql.Rows) ([]Repo, error) {
	var out []Repo
	for rows.Next() {
		var repo Repo
		var autoPublish, active int
		var created string
		if err := rows.Scan(&repo.ID, &repo.Name, &repo.URL, &repo.Platform, &repo.DefaultBranch, &autoPublish, &repo.PublishMode, &active, &created); err != nil {
			return nil, err
		}
		repo.AutoPublish = autoPublish == 1
		repo.Active = active == 1
		repo.CreatedAt = parseDBTime(created)
		out = append(out, repo)
	}
	return out, rows.Err()
}

func scanMemory(rows *sql.Rows) ([]MemoryEntry, error) {
	var out []MemoryEntry
	for rows.Next() {
		var entry MemoryEntry
		var created, updated string
		var expires sql.NullString
		if err := rows.Scan(&entry.ID, &entry.RepoID, &entry.Type, &entry.Key, &entry.Content, &entry.Dimension, &entry.SourceMR, &created, &updated, &expires); err != nil {
			return nil, err
		}
		entry.CreatedAt = parseDBTime(created)
		entry.UpdatedAt = parseDBTime(updated)
		if expires.Valid && expires.String != "" {
			t := parseDBTime(expires.String)
			entry.ExpiresAt = &t
		}
		out = append(out, entry)
	}
	return out, rows.Err()
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func nullable(s string) any {
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

func parseDBTime(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	if t.IsZero() {
		t, _ = time.Parse("2006-01-02 15:04:05", s)
	}
	return t
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
