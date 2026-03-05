CREATE TABLE IF NOT EXISTS users (
  id          BIGINT PRIMARY KEY,
  username    VARCHAR(255),
  first_name  VARCHAR(255),
  last_name   VARCHAR(255),
  timezone    VARCHAR(64) NOT NULL DEFAULT 'Asia/Jakarta',
  created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS raw_messages (
  id          BIGINT AUTO_INCREMENT PRIMARY KEY,
  user_id     BIGINT NOT NULL,
  chat_id     BIGINT NOT NULL,
  message_id  INT NOT NULL,
  text        TEXT NOT NULL,
  created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_raw_messages_user_id (user_id),
  INDEX idx_raw_messages_created_at (created_at)
);

CREATE TABLE IF NOT EXISTS entries (
  id          BIGINT AUTO_INCREMENT PRIMARY KEY,
  user_id     BIGINT NOT NULL,
  chat_id     BIGINT NOT NULL,
  message_id  INT NOT NULL,
  timestamp   DATETIME NOT NULL,
  amount      BIGINT NOT NULL,
  currency    VARCHAR(8) NOT NULL DEFAULT 'IDR',
  type        ENUM('expense','income','transfer','unknown') NOT NULL DEFAULT 'unknown',
  category    VARCHAR(128) NOT NULL DEFAULT 'General',
  description VARCHAR(512) NOT NULL DEFAULT '',
  raw_text    TEXT NOT NULL,
  confidence  DOUBLE NOT NULL DEFAULT 0.0,
  created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_entries_user_id (user_id),
  INDEX idx_entries_timestamp (timestamp),
  INDEX idx_entries_user_timestamp (user_id, timestamp),
  INDEX idx_entries_message (user_id, message_id)
);
