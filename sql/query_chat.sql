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
WHERE id = ?;

-- name: createChatStatDaily :one
INSERT INTO chat_stat_daily (chat_id, stat_date)
VALUES (?, ?)
RETURNING *;

-- name: UpdateChatStatDaily :exec
UPDATE chat_stat_daily
SET message_count        = ?,
    photo_count          = ?,
    video_count          = ?,
    sticker_count        = ?,
    forward_count        = ?,
    mars_count           = ?,
    max_mars_count       = ?,
    racy_count           = ?,
    adult_count          = ?,
    download_video_count = ?,
    download_audio_count = ?,
    dio_add_user_count   = ?,
    dio_ban_user_count   = ?,
    user_msg_stat        = ?,
    msg_count_by_time    = ?,
    msg_id_at_time_start = ?
WHERE chat_id = ?
  AND stat_date = ?;

-- name: getChatStat :one
SELECT *
FROM chat_stat_daily
WHERE chat_id = ?
  AND stat_date = ?;