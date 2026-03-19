ALTER TABLE messages
  DROP INDEX uk_messages_client_msg_id;

ALTER TABLE messages
  ADD UNIQUE KEY uk_messages_sender_client_msg_id (sender_id, client_msg_id);
