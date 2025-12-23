-- encoding: utf-8

-- name: GetBiliInlineData :one
SELECT text, chat_id, msg_id
FROM bili_inline_results
WHERE uid = $1;


-- name: CreateBiliInlineData :one
INSERT INTO bili_inline_results
    DEFAULT
VALUES
RETURNING uid;

-- name: UpdateBiliInlineMsgId :exec
UPDATE bili_inline_results
SET text    = $1,
    chat_id = $2,
    msg_id  = $3
WHERE uid = $4;
