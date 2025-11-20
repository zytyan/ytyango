-- encoding: utf-8

-- name: GetBiliInlineData :one
SELECT text, chat_id, msg_id
FROM bili_inline_results
WHERE uid = ?;


-- name: InsertBiliInlineData :one
INSERT INTO bili_inline_results
    DEFAULT
VALUES
/*这里只插入一个uid，并由数据库返回，因为chat id和msg id只有发出去了之后才知道*/
RETURNING uid;

-- name: UpdateBiliInlineMsgId :exec
UPDATE bili_inline_results
SET text    = ?,
    chat_id = ?,
    msg_id  = ?
WHERE uid = ?;