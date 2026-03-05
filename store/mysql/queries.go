package mysql

const (
	qUpsertUser = `
INSERT INTO users (id, username, first_name, last_name, timezone)
VALUES (?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
  username = VALUES(username),
  first_name = VALUES(first_name),
  last_name = VALUES(last_name),
  timezone = VALUES(timezone),
  updated_at = CURRENT_TIMESTAMP`

	qGetUser = `
SELECT id, username, first_name, last_name, timezone, created_at, updated_at
FROM users
WHERE id = ?`

	qSaveMessage = `
INSERT INTO raw_messages (user_id, chat_id, message_id, text, created_at)
VALUES (?, ?, ?, ?, ?)`

	qSaveEntry = `
INSERT INTO entries
(user_id, chat_id, message_id, timestamp, amount, currency, type, category, description, raw_text, confidence)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	qUpdateEntry = `
UPDATE entries
SET amount = ?, currency = ?, type = ?, category = ?, description = ?, raw_text = ?, confidence = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?`

	qDeleteEntry = `DELETE FROM entries WHERE id = ?`

	qGetEntryByID = `
SELECT id, user_id, chat_id, message_id, timestamp, amount, currency, type, category, description, raw_text, confidence, created_at
FROM entries
WHERE id = ?`

	qGetEntryByMessageID = `
SELECT id, user_id, chat_id, message_id, timestamp, amount, currency, type, category, description, raw_text, confidence, created_at
FROM entries
WHERE user_id = ? AND message_id = ?
ORDER BY id DESC
LIMIT 1`

	qGetLastEntries = `
SELECT id, user_id, chat_id, message_id, timestamp, amount, currency, type, category, description, raw_text, confidence, created_at
FROM entries
WHERE user_id = ?
ORDER BY timestamp DESC
LIMIT ?`

	qGetEntriesByDateRange = `
SELECT id, user_id, chat_id, message_id, timestamp, amount, currency, type, category, description, raw_text, confidence, created_at
FROM entries
WHERE user_id = ? AND timestamp >= ? AND timestamp < ?
ORDER BY timestamp DESC`

	qGetCategoryTotals = `
SELECT category, SUM(amount) AS total
FROM entries
WHERE user_id = ? AND timestamp >= ? AND timestamp < ?
GROUP BY category`
)
