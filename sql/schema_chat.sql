CREATE TABLE IF NOT EXISTS chat_cfg
(
    id               INTEGER PRIMARY KEY NOT NULL,
    web_id           INTEGER, -- Nullable, group web id 可能没有配置
    auto_cvt_bili    INT_BOOL            NOT NULL CHECK ( auto_cvt_bili in (0, 1)),
    auto_ocr         INT_BOOL            NOT NULL CHECK ( auto_ocr in (0, 1)),
    auto_calculate   INT_BOOL            NOT NULL CHECK ( auto_calculate in (0, 1)),
    auto_exchange    INT_BOOL            NOT NULL CHECK ( auto_exchange in (0, 1)),
    auto_check_adult INT_BOOL            NOT NULL CHECK ( auto_check_adult in (0, 1)),
    save_messages    INT_BOOL            NOT NULL CHECK ( save_messages in (0, 1)),
    enable_coc       INT_BOOL            NOT NULL CHECK ( enable_coc in (0, 1)),
    resp_nsfw_msg    INT_BOOL            NOT NULL CHECK ( resp_nsfw_msg in (0, 1))
);

CREATE INDEX IF NOT EXISTS idx_chat_cfg
    ON chat_cfg (web_id);