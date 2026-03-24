ALTER TABLE sync_events
  ADD COLUMN aggregate_id VARCHAR(128) NOT NULL DEFAULT '' AFTER event_type;

CREATE INDEX idx_sync_events_user_aggregate ON sync_events (user_id, aggregate_id, id);
