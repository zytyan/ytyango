-- encoding: utf-8

-- name: CreateNewGeminiSession :one
INSERT INTO gemini_sessions (chat_id, chat_name, chat_type)
VALUES (?, ?, ?)
RETURNING *;

-- name: AddGeminiMessage :exec
INSERT INTO gemini_contents (session_id,
                             chat_id,
                             msg_id,
                             role,
                             sent_time,
                             username,
                             msg_type,
                             reply_to_msg_id,
                             text,
                             blob,
                             mime_type,
                             quote_part,
                             thought_signature)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: getAllMsgInSessionReversed :many
SELECT *
FROM gemini_contents
WHERE session_id = ?
ORDER BY msg_id DESC
LIMIT ?;

-- name: GetSessionIdByMessage :one
SELECT gemini_contents.session_id
FROM gemini_contents
WHERE chat_id = ?
  AND msg_id = ?;

-- name: GetSessionById :one
SELECT *
FROM gemini_sessions
WHERE id = ?;