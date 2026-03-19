ALTER TABLE conversations
  ADD COLUMN announcement VARCHAR(512) NOT NULL DEFAULT '' AFTER name;

CREATE TABLE IF NOT EXISTS conversation_settings (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  conversation_id BIGINT UNSIGNED NOT NULL,
  user_id BIGINT UNSIGNED NOT NULL,
  is_muted TINYINT(1) NOT NULL DEFAULT 0,
  draft TEXT NOT NULL,
  pinned_at TIMESTAMP NULL DEFAULT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  deleted_at TIMESTAMP NULL DEFAULT NULL,
  UNIQUE KEY uk_conversation_settings_pair (conversation_id, user_id),
  KEY idx_conversation_settings_user (user_id, pinned_at),
  CONSTRAINT fk_conversation_settings_conversation FOREIGN KEY (conversation_id) REFERENCES conversations(id),
  CONSTRAINT fk_conversation_settings_user FOREIGN KEY (user_id) REFERENCES users(id)
);
