-- encoding: utf-8

-- name: getUserById :one
SELECT *
FROM users
WHERE user_id = ?;

-- name: getChatById :one
SELECT *
FROM chat_cfg
WHERE id = ?;

-- name: getChatIdByWebId :one
SELECT id
FROM chat_cfg
WHERE web_id = ?;

-- name: CreateNewChatDefaultCfg :one
INSERT INTO chat_cfg (id, auto_cvt_bili, auto_ocr, auto_calculate, auto_exchange, auto_check_adult,
                      save_messages, enable_coc, resp_nsfw_msg)
VALUES (?,
        FALSE,
        FALSE,
        FALSE,
        FALSE,
        FALSE,
        TRUE,
        FALSE,
        FALSE)
RETURNING *;