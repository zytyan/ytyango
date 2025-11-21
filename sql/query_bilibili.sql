-- encoding: utf-8

-- name: GetBiliInlineData :one
SELECT text, chat_id, msg_id
FROM bili_inline_results
WHERE uid = ?;


-- name: InsertBiliInlineData :one
INSERT INTO bili_inline_results
    DEFAULT
VALUES
RETURNING uid;

-- name: UpdateBiliInlineMsgId :exec
UPDATE bili_inline_results
SET text    = ?,
    chat_id = ?,
    msg_id  = ?
WHERE uid = ?;