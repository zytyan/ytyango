-- encoding: utf-8

-- name: CreateGeminiSession :one
INSERT INTO gemini_sessions (chat_id, starter_id, root_msg_id, started_at, last_active_at)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: GetGeminiSessionByID :one
SELECT *
FROM gemini_sessions
WHERE id = ?;

-- name: TouchGeminiSession :exec
UPDATE gemini_sessions
SET last_active_at = ?
WHERE id = ?;

-- name: GetGeminiSessionByTgMsg :one
SELECT s.*
FROM gemini_sessions s
         JOIN gemini_messages m ON m.session_id = s.id
WHERE m.chat_id = ?
  AND m.tg_message_id = ?
LIMIT 1;

-- name: GetGeminiMessageByTgMsg :one
SELECT *
FROM gemini_messages
WHERE chat_id = ?
  AND tg_message_id = ?
LIMIT 1;

-- name: GetGeminiLastSeq :one
SELECT CAST(COALESCE(MAX(seq), 0) AS INTEGER) AS last_seq
FROM gemini_messages
WHERE session_id = ?;

-- name: ListGeminiMessages :many
SELECT *
FROM gemini_messages
WHERE session_id = ?
ORDER BY seq DESC
LIMIT ?;

-- name: CreateGeminiMessage :one
INSERT INTO gemini_messages (session_id, chat_id, tg_message_id, from_id, role, content, seq, reply_to_seq,
                             created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;
