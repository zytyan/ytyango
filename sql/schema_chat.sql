CREATE TABLE IF NOT EXISTS chat_cfg
(
    id               bigint   PRIMARY KEY NOT NULL,
    web_id           bigint, -- Nullable, group web id 可能没有配置
    auto_cvt_bili    boolean  NOT NULL,
    auto_ocr         boolean  NOT NULL,
    auto_calculate   boolean  NOT NULL,
    auto_exchange    boolean  NOT NULL,
    auto_check_adult boolean  NOT NULL,
    save_messages    boolean  NOT NULL,
    enable_coc       boolean  NOT NULL,
    resp_nsfw_msg    boolean  NOT NULL,
    timezone         integer  NOT NULL CHECK ( timezone < 86400 AND timezone > -86400)
);

CREATE INDEX IF NOT EXISTS idx_chat_cfg
    ON chat_cfg (web_id);


CREATE TABLE chat_stat_daily
(
    chat_id              bigint               NOT NULL,
    stat_date            integer              NOT NULL, -- 从Unix纪元开始的日期数量

    message_count        bigint               NOT NULL DEFAULT 0,
    photo_count          bigint               NOT NULL DEFAULT 0,
    video_count          bigint               NOT NULL DEFAULT 0,
    sticker_count        bigint               NOT NULL DEFAULT 0,
    forward_count        bigint               NOT NULL DEFAULT 0,

    mars_count           bigint               NOT NULL DEFAULT 0,
    max_mars_count       bigint               NOT NULL DEFAULT 0,

    racy_count           bigint               NOT NULL DEFAULT 0,
    adult_count          bigint               NOT NULL DEFAULT 0,

    download_video_count bigint               NOT NULL DEFAULT 0,
    download_audio_count bigint               NOT NULL DEFAULT 0,

    dio_add_user_count   bigint               NOT NULL DEFAULT 0,
    dio_ban_user_count   bigint               NOT NULL DEFAULT 0,

    -- serialized MessagePack
    user_msg_stat        bytea                NOT NULL DEFAULT '\x'::bytea,
    msg_count_by_time    bytea                NOT NULL DEFAULT '\x'::bytea,
    msg_id_at_time_start bytea                NOT NULL DEFAULT '\x'::bytea,

    PRIMARY KEY (chat_id, stat_date)
);
