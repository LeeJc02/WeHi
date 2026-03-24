-- Reserved for future legacy-alignment migrations.
-- The current runtime verification uses a clean MySQL schema created by 001_initial_schema.sql.

CREATE TABLE IF NOT EXISTS sync_events (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  user_id BIGINT UNSIGNED NOT NULL,
  event_type VARCHAR(64) NOT NULL,
  payload JSON NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_sync_events_user_cursor (user_id, id),
  CONSTRAINT fk_sync_events_user FOREIGN KEY (user_id) REFERENCES users(id)
);
