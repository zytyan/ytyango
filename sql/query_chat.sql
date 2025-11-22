-- encoding: utf-8

-- name: getChatById :one
SELECT *
FROM chat_cfg
WHERE id = ?;

-- name: getChatIdByWebId :one
SELECT id
FROM chat_cfg
WHERE web_id = ?;

-- name: CreateNewChatDefaultCfg :one
INSERT INTO chat_cfg (id, web_id, auto_cvt_bili, auto_ocr, auto_calculate, auto_exchange, auto_check_adult,
                      save_messages, enable_coc, resp_nsfw_msg)
VALUES (?,
        NULL,
        FALSE,
        FALSE,
        FALSE,
        FALSE,
        FALSE,
        TRUE,
        FALSE,
        FALSE)
RETURNING *;

-- name: updateChat :exec
UPDATE chat_cfg
SET auto_cvt_bili=?,
    auto_ocr=?,
    auto_calculate=?,
    auto_exchange=?,
    auto_check_adult=?,
    save_messages=?,
    enable_coc=?,
    resp_nsfw_msg=?
WHERE id = ?