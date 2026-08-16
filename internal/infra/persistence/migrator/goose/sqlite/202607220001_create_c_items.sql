-- +goose Up
CREATE TABLE IF NOT EXISTS c_items (
    id BIGINT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    content_type VARCHAR(16) NOT NULL,
    title VARCHAR(255) NOT NULL DEFAULT '',
    body TEXT NOT NULL DEFAULT '',
    lifecycle VARCHAR(16) NOT NULL,
    importance VARCHAR(16) NOT NULL,
    source VARCHAR(32) NOT NULL DEFAULT 'web',
    archived_at DATETIME NULL,
    trashed_at DATETIME NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_c_items_user_created ON c_items (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_c_items_user_lifecycle_created ON c_items (user_id, lifecycle, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_c_items_user_importance_lifecycle ON c_items (user_id, importance, lifecycle);
CREATE INDEX IF NOT EXISTS idx_c_items_pending_created ON c_items (lifecycle, created_at);
CREATE INDEX IF NOT EXISTS idx_c_items_trash_trashed ON c_items (lifecycle, trashed_at);

CREATE TABLE IF NOT EXISTS c_item_attachments (
    id BIGINT PRIMARY KEY,
    item_id BIGINT NOT NULL,
    upload_id BIGINT NOT NULL,
    sort INT NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_c_item_attachments_item ON c_item_attachments (item_id);
CREATE INDEX IF NOT EXISTS idx_c_item_attachments_upload ON c_item_attachments (upload_id);

INSERT INTO w_system_configs (key, value, type, visibility, description, created_at, updated_at)
VALUES
  ('item_pending_archive_after_days', '3', 'system', 0, '未处理 Item 自动归档天数', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  ('item_trash_purge_after_days', '30', 'system', 0, '垃圾箱 Item 彻底删除天数', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (key) DO NOTHING;

INSERT INTO w_schedules (id, name, task_type, cron, payload, is_active, created_at, updated_at)
VALUES
  (2, 'Item 未处理归档', 'item_archive_pending', '15 * * * *', '{}', TRUE, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  (3, 'Item 垃圾箱清理', 'item_purge_trash', '45 * * * *', '{}', TRUE, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (id) DO NOTHING;

-- +goose Down
DELETE FROM w_schedules WHERE id IN (2, 3);
DELETE FROM w_system_configs WHERE key IN ('item_pending_archive_after_days', 'item_trash_purge_after_days');
DROP TABLE IF EXISTS c_item_attachments;
DROP TABLE IF EXISTS c_items;
