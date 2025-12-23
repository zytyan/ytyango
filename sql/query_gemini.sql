-- encoding: utf-8

-- name: CreateNewGeminiSession :one
INSERT INTO gemini_sessions (chat_id, chat_name, chat_type)
VALUES ($1, $2, $3)
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
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13);

-- name: getAllMsgInSessionReversed :many
SELECT *
FROM gemini_contents
WHERE session_id = $1
ORDER BY msg_id DESC
LIMIT $2;

-- name: GetSessionIdByMessage :one
SELECT gemini_contents.session_id
FROM gemini_contents
WHERE chat_id = $1
  AND msg_id = $2;

-- name: GetSessionById :one
SELECT *
FROM gemini_sessions
WHERE id = $1;
