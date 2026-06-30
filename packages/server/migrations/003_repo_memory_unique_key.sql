DELETE FROM repo_memory
WHERE rowid NOT IN (
    SELECT MIN(rowid)
    FROM repo_memory
    GROUP BY repo_id, type, key
);

CREATE UNIQUE INDEX idx_repo_memory_repo_type_key ON repo_memory(repo_id, type, key);
