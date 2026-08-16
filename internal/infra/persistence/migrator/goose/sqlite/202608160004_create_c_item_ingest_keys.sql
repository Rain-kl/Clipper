-- +goose Up
CREATE TABLE IF NOT EXISTS c_item_ingest_keys (
    channel_id BIGINT NOT NULL,
    message_id VARCHAR(128) NOT NULL,
    user_id BIGINT NOT NULL,
    item_id BIGINT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (channel_id, message_id)
);

-- +goose Down
DROP TABLE IF EXISTS c_item_ingest_keys;
