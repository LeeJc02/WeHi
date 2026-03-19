ALTER TABLE users
  ADD COLUMN avatar_url VARCHAR(512) NOT NULL DEFAULT '' AFTER display_name;

ALTER TABLE friendships
  ADD COLUMN remark_name VARCHAR(128) NOT NULL DEFAULT '' AFTER friend_id;
