-- encoding: utf-8

CREATE TABLE IF NOT EXISTS gemini_sessions
(
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    chat_id        INTEGER      NOT NULL,
    starter_id     INTEGER      NOT NULL,
    root_msg_id    INTEGER      NOT NULL,
    started_at     INT_UNIX_SEC NOT NULL,
    last_active_at INT_UNIX_SEC NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_gemini_sessions_chat
    ON gemini_sessions (chat_id, last_active_at);

CREATE TABLE IF NOT EXISTS gemini_messages
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

CREATE INDEX IF NOT EXISTS idx_gemini_messages_session_seq
    ON gemini_messages (session_id, seq);

CREATE INDEX IF NOT EXISTS idx_gemini_messages_chat_tg
    ON gemini_messages (chat_id, tg_message_id);
