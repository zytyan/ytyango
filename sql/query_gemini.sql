
-- name: CreateGeminiSession :one
INSERT INTO gemini_sessions (chat_id, chat_name, chat_type, tools, cache_name, cache_ttl, cache_expired)
VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdateGeminiSessionCache :exec
UPDATE gemini_sessions
SET tools         = ?,
    cache_name    = ?,
    cache_ttl     = ?,
    cache_expired = ?
WHERE id = ?;

-- name: GetGeminiSessionByChat :one
SELECT *
FROM gemini_sessions
WHERE chat_id = ?
ORDER BY id DESC
LIMIT 1;

-- name: GetGeminiSessionById :one
SELECT *
FROM gemini_sessions
WHERE id = ?;

-- name: GetNextGeminiSeq :one
SELECT COALESCE(MAX(seq), 0) + 1
FROM gemini_content_v2
WHERE session_id = ?;

-- name: AddGeminiContentV2 :one
INSERT INTO gemini_content_v2 (session_id, role, seq, x_user_extra)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: AddGeminiContentV2Part :exec
INSERT INTO gemini_content_v2_parts (
    content_id,
    part_index,
    text,
    thought,
    thought_signature,
    inline_data,
    inline_data_mime,
    file_uri,
    file_mime,
    function_call_name,
    function_call_args,
    function_response_name,
    function_response,
    executable_code,
    executable_code_language,
    code_execution_outcome,
    code_execution_output,
    video_start_offset,
    video_end_offset,
    video_fps,
    x_user_extra
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: ListGeminiContentV2 :many
SELECT *
FROM gemini_content_v2
WHERE session_id = ?
ORDER BY seq ASC
LIMIT ?;

-- name: ListGeminiContentV2Parts :many
SELECT *
FROM gemini_content_v2_parts
WHERE content_id = ?
ORDER BY part_index ASC;
