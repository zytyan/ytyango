-- Sessions（会话表）
CREATE TABLE gemini_sessions
(
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    chat_id   INTEGER NOT NULL,
    chat_name TEXT    NOT NULL,
    chat_type TEXT    NOT NULL
) STRICT;

-- Contents（消息内容表）
CREATE TABLE gemini_contents
(
    session_id        INTEGER      NOT NULL,
    chat_id           INTEGER      NOT NULL,
    msg_id            INTEGER      NOT NULL, -- 对应 MsgId
    role              TEXT         NOT NULL,
    sent_time         INT_UNIX_SEC NOT NULL, -- yyyy-mm-dd HH:MM:SS
    username          TEXT         NOT NULL,
    msg_type          TEXT         NOT NULL, -- 使用英语标识类型，包括 text, photo, sticker，将来可能有更多类型（或许）
    reply_to_msg_id   INTEGER,               -- 若有，代表该消息为回复消息
    text              TEXT,                  -- 可以与blob共存，若同时存在，则使用两个part，但两个至少应该有一个
    blob              BLOB,
    mime_type         TEXT,                  -- 若blob存在，mime_type必须存在
    quote_part        TEXT,                  -- 回复消息时，被回复的消息被引用的部分。
    thought_signature TEXT,                  -- 模型的思考签名
    -- 一个消息唯一由 SessionId + MsgId 组成
    PRIMARY KEY (session_id, msg_id),

    -- 外键指向 gemini_sessions
    FOREIGN KEY (session_id)
        REFERENCES gemini_sessions (id)
        ON DELETE CASCADE,
    UNIQUE (chat_id, msg_id),
    CHECK ( text IS NOT NULL OR blob IS NOT NULL ),
    CHECK (
        (blob IS NULL AND mime_type IS NULL)
            OR
        (blob IS NOT NULL AND mime_type IS NOT NULL)
        )
) WITHOUT ROWID;