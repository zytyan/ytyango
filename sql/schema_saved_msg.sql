-- encoding: utf-8

CREATE TABLE IF NOT EXISTS saved_msgs
(
    message_id          INTEGER      NOT NULL,
    chat_id             INTEGER      NOT NULL,
    from_user_id        INTEGER,
    sender_chat_id      INTEGER,
    date                INT_UNIX_SEC NOT NULL,
    forward_origin_name TEXT,
    forward_origin_id   INTEGER,
    message_thread_id   INTEGER,
    reply_to_message_id INTEGER,
    reply_to_chat_id    INTEGER,
    via_bot_id          INTEGER,
    edit_date           INT_UNIX_SEC,
    media_group_id      TEXT,
    text                TEXT,
    entities_json       BLOB_JSONB,

    media_id            TEXT,
    media_uid           TEXT,
    -- photo, video, sticker, story, video_note, voice, ...
    media_type          TEXT,

    extra_data          BLOB_JSONB,
    extra_type          TEXT,
    -- RAW_UPDATE_JSON 放入单独的表，避免单表过大
    PRIMARY KEY (chat_id, message_id)
) WITHOUT ROWID;

CREATE TABLE IF NOT EXISTS raw_update
(
    id         INTEGER NOT NULL PRIMARY KEY,
    chat_id    INTEGER,
    message_id INTEGER,
    raw_update BLOB_JSONB
);

CREATE INDEX IF NOT EXISTS idx_raw_update_chat_message_id
    ON raw_update (id, chat_id);

CREATE TABLE IF NOT EXISTS edit_history
(
    chat_id    INTEGER NOT NULL,
    message_id INTEGER NOT NULL,
    edit_id    INTEGER NOT NULL,
    text       TEXT    NOT NULL,
    PRIMARY KEY (chat_id, message_id, edit_id)
) WITHOUT ROWID;

CREATE TRIGGER IF NOT EXISTS trigger_on_edit_message
    AFTER UPDATE
    ON saved_msgs
BEGIN
    INSERT INTO edit_history (chat_id, message_id, edit_id, text)
    VALUES (OLD.chat_id,
            OLD.message_id,
            COALESCE((SELECT MAX(e.edit_id) + 1
                      FROM edit_history AS e
                      WHERE e.chat_id = OLD.chat_id
                        AND e.message_id = OLD.message_id), 1),
            OLD.text);
END;