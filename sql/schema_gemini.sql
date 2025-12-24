-- Sessions（会话表）
CREATE TABLE gemini_sessions
(
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    chat_id   INTEGER NOT NULL,
    chat_name TEXT    NOT NULL,
    chat_type TEXT    NOT NULL,
    frozen    INTEGER NOT NULL DEFAULT 0
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

-- Messages（上下文消息表）
CREATE TABLE gemini_messages
(
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id    INTEGER      NOT NULL REFERENCES gemini_sessions (id) ON DELETE CASCADE,
    chat_id       INTEGER      NOT NULL,
    tg_message_id INTEGER      NOT NULL,
    from_id       INTEGER      NOT NULL,
    role          TEXT         NOT NULL,
    content       TEXT         NOT NULL,
    seq           INTEGER      NOT NULL,
    reply_to_seq  INTEGER,
    created_at    INT_UNIX_SEC NOT NULL,
    UNIQUE (chat_id, tg_message_id),
    UNIQUE (session_id, seq)
);

CREATE INDEX idx_gemini_messages_session_seq
    ON gemini_messages (session_id, seq);

CREATE INDEX idx_gemini_messages_chat_tg
    ON gemini_messages (chat_id, tg_message_id);

-- Session Migrations（会话合并/迁移记录）
CREATE TABLE gemini_session_migrations
(
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    old_session_id   INTEGER      NOT NULL,
    new_session_id   INTEGER      NOT NULL,
    migrated_msg_ids TEXT         NOT NULL, -- comma separated msg_id list
    reason           TEXT,
    requested_by     TEXT,
    created_at       INT_UNIX_SEC NOT NULL,
    FOREIGN KEY (old_session_id) REFERENCES gemini_sessions (id),
    FOREIGN KEY (new_session_id) REFERENCES gemini_sessions (id)
);

-- Memories（记忆表）
CREATE TABLE gemini_memories
(
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    chat_id    INTEGER      NOT NULL,
    user_id    INTEGER,
    type       TEXT         NOT NULL,
    content    TEXT         NOT NULL,
    importance INTEGER      NOT NULL DEFAULT 0,
    expires_at INT_UNIX_SEC,
    created_at INT_UNIX_SEC NOT NULL,
    updated_at INT_UNIX_SEC NOT NULL,
    created_by TEXT
);

CREATE INDEX idx_gemini_memories_chat_user ON gemini_memories (chat_id, user_id);
CREATE INDEX idx_gemini_memories_expires ON gemini_memories (expires_at);

-- Content V2 (structured genai content/parts)
CREATE TABLE IF NOT EXISTS gemini_content_v2
(
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id   INTEGER      NOT NULL REFERENCES gemini_sessions (id) ON DELETE CASCADE,
    role         TEXT         NOT NULL,
    seq          INTEGER      NOT NULL,
    x_user_extra JSON_TEXT,
    UNIQUE (session_id, seq)
);

CREATE INDEX IF NOT EXISTS idx_gemini_content_v2_session ON gemini_content_v2 (session_id);

CREATE TABLE IF NOT EXISTS gemini_content_v2_parts
(
    id                        INTEGER PRIMARY KEY AUTOINCREMENT,
    content_id                INTEGER      NOT NULL REFERENCES gemini_content_v2 (id) ON DELETE CASCADE,
    part_index                INTEGER      NOT NULL,
    text                      TEXT,
    thought                   INT_BOOL     NOT NULL DEFAULT 0 CHECK (thought IN (0, 1)),
    thought_signature         BLOB,
    inline_data               BLOB,
    inline_data_mime          TEXT,
    file_uri                  TEXT,
    file_mime                 TEXT,
    function_call_name        TEXT,
    function_call_args        JSON_TEXT,
    function_response_name    TEXT,
    function_response         JSON_TEXT,
    executable_code           TEXT,
    executable_code_language  TEXT,
    code_execution_outcome    TEXT,
    code_execution_output     TEXT,
    video_start_offset        TEXT,
    video_end_offset          TEXT,
    video_fps                 REAL,
    x_user_extra              JSON_TEXT,
    UNIQUE (content_id, part_index)
);

CREATE INDEX IF NOT EXISTS idx_gemini_content_v2_parts_content ON gemini_content_v2_parts (content_id);
