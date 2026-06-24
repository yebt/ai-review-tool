CREATE TABLE repos (
    id              TEXT PRIMARY KEY,
    name            TEXT NOT NULL UNIQUE,
    url             TEXT NOT NULL,
    platform        TEXT NOT NULL,
    default_branch  TEXT NOT NULL DEFAULT 'main',
    webhook_secret  TEXT,
    auto_publish    INTEGER NOT NULL DEFAULT 0,
    publish_mode    TEXT NOT NULL DEFAULT 'sequential',
    active          INTEGER NOT NULL DEFAULT 1,
    created_at      TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_repos_platform_active ON repos(platform, active);

CREATE TABLE model_configs (
    id           TEXT PRIMARY KEY,
    repo_id      TEXT NOT NULL REFERENCES repos(id) ON DELETE CASCADE,
    provider     TEXT NOT NULL,
    model_name   TEXT NOT NULL,
    api_key_env  TEXT,
    extra_params TEXT,
    is_active    INTEGER NOT NULL DEFAULT 1,
    created_at   TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_model_configs_repo_active ON model_configs(repo_id, is_active);

CREATE TABLE channels (
    id     TEXT PRIMARY KEY,
    type   TEXT NOT NULL,
    name   TEXT NOT NULL UNIQUE,
    config TEXT NOT NULL,
    active INTEGER NOT NULL DEFAULT 1
);

CREATE INDEX idx_channels_type_active ON channels(type, active);

CREATE TABLE repo_channels (
    repo_id    TEXT NOT NULL REFERENCES repos(id) ON DELETE CASCADE,
    channel_id TEXT NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    events     TEXT NOT NULL DEFAULT '["review.generated","review.published"]',
    PRIMARY KEY (repo_id, channel_id)
);

CREATE INDEX idx_repo_channels_channel ON repo_channels(channel_id);

CREATE TABLE skills (
    id        TEXT PRIMARY KEY,
    name      TEXT NOT NULL UNIQUE,
    dimension TEXT NOT NULL,
    file_path TEXT NOT NULL,
    active    INTEGER NOT NULL DEFAULT 1,
    loaded_at TEXT
);

CREATE INDEX idx_skills_dimension_active ON skills(dimension, active);

CREATE TABLE repo_skills (
    repo_id  TEXT NOT NULL REFERENCES repos(id) ON DELETE CASCADE,
    skill_id TEXT NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
    priority INTEGER NOT NULL DEFAULT 100,
    PRIMARY KEY (repo_id, skill_id)
);

CREATE INDEX idx_repo_skills_skill ON repo_skills(skill_id);

CREATE TABLE reviews (
    id           TEXT PRIMARY KEY,
    repo_id      TEXT NOT NULL REFERENCES repos(id),
    mr_id        TEXT NOT NULL,
    mr_url       TEXT,
    mr_title     TEXT,
    base_sha     TEXT,
    head_sha     TEXT,
    start_sha    TEXT,
    model_used   TEXT NOT NULL,
    status       TEXT NOT NULL DEFAULT 'pending',
    auto_publish INTEGER NOT NULL DEFAULT 0,
    scores       TEXT,
    verdict      TEXT,
    created_at   TEXT NOT NULL DEFAULT (datetime('now')),
    completed_at TEXT
);

CREATE INDEX idx_reviews_repo_created ON reviews(repo_id, created_at);
CREATE INDEX idx_reviews_repo_mr ON reviews(repo_id, mr_id);
CREATE INDEX idx_reviews_status ON reviews(status);

CREATE TABLE review_comments (
    id                  TEXT PRIMARY KEY,
    review_id           TEXT NOT NULL REFERENCES reviews(id) ON DELETE CASCADE,
    dimension           TEXT NOT NULL,
    severity            TEXT NOT NULL,
    file                TEXT NOT NULL,
    line_start          INTEGER,
    line_end            INTEGER,
    evidence            TEXT NOT NULL,
    why                 TEXT NOT NULL,
    suggestion_snippet  TEXT,
    status              TEXT NOT NULL DEFAULT 'pending',
    platform_comment_id TEXT,
    created_at          TEXT NOT NULL DEFAULT (datetime('now')),
    published_at        TEXT
);

CREATE INDEX idx_review_comments_review_status ON review_comments(review_id, status);
CREATE INDEX idx_review_comments_dimension_severity ON review_comments(dimension, severity);

CREATE TABLE repo_memory (
    id         TEXT PRIMARY KEY,
    repo_id    TEXT NOT NULL REFERENCES repos(id) ON DELETE CASCADE,
    type       TEXT NOT NULL,
    key        TEXT NOT NULL,
    content    TEXT NOT NULL,
    dimension  TEXT,
    source_mr  TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    expires_at TEXT
);

CREATE INDEX idx_repo_memory_repo_type ON repo_memory(repo_id, type);
CREATE INDEX idx_repo_memory_repo_dimension ON repo_memory(repo_id, dimension);
