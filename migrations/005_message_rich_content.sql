ALTER TABLE messages
  ADD COLUMN reply_to_message_id BIGINT UNSIGNED NULL AFTER content,
  ADD COLUMN attachment_json TEXT NULL AFTER reply_to_message_id,
  ADD COLUMN delivery_status VARCHAR(32) NOT NULL DEFAULT 'sent' AFTER status,
  ADD COLUMN recalled_at TIMESTAMP NULL DEFAULT NULL AFTER updated_at,
  ADD KEY idx_messages_reply_to_message_id (reply_to_message_id);
