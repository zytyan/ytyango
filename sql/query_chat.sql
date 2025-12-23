-- encoding: utf-8

-- name: getChatCfgById :one
SELECT *
FROM chat_cfg
WHERE id = $1;

-- name: getChatIdByWebId :one
SELECT id
FROM chat_cfg
WHERE web_id = $1;

-- name: CreateChatCfg :exec
INSERT INTO chat_cfg (id, web_id, auto_cvt_bili, auto_ocr, auto_calculate, auto_exchange, auto_check_adult,
                      save_messages, enable_coc, resp_nsfw_msg, timezone)
VALUES ($1, $2, $3, $4, $5, $6,
        $7, $8, $9, $10, $11);

-- name: updateChatCfg :exec
UPDATE chat_cfg
SET auto_cvt_bili=$1,
    auto_ocr=$2,
    auto_calculate=$3,
    auto_exchange=$4,
    auto_check_adult=$5,
    save_messages=$6,
    enable_coc=$7,
    resp_nsfw_msg=$8
WHERE id = $9;

-- name: createChatStatDaily :one
INSERT INTO chat_stat_daily (chat_id, stat_date)
VALUES ($1, $2)
RETURNING *;

-- name: UpdateChatStatDaily :exec
UPDATE chat_stat_daily
SET message_count        = $1,
    photo_count          = $2,
    video_count          = $3,
    sticker_count        = $4,
    forward_count        = $5,
    mars_count           = $6,
    max_mars_count       = $7,
    racy_count           = $8,
    adult_count          = $9,
    download_video_count = $10,
    download_audio_count = $11,
    dio_add_user_count   = $12,
    dio_ban_user_count   = $13,
    user_msg_stat        = $14,
    msg_count_by_time    = $15,
    msg_id_at_time_start = $16
WHERE chat_id = $17
  AND stat_date = $18;

-- name: getChatStat :one
SELECT *
FROM chat_stat_daily
WHERE chat_stat_daily.chat_id = $1
  AND chat_stat_daily.stat_date = $2;
