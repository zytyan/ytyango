-- encoding: utf-8

CREATE TABLE IF NOT EXISTS saved_msgs
(
    message_id          bigint       NOT NULL,
    chat_id             bigint       NOT NULL,
    from_user_id        bigint,
    sender_chat_id      bigint,
    date                timestamptz  NOT NULL,
    forward_origin_name TEXT,
    forward_origin_id   bigint,
    message_thread_id   bigint,
    reply_to_message_id bigint,
    reply_to_chat_id    bigint,
    via_bot_id          bigint,
    edit_date           timestamptz,
    media_group_id      TEXT,
    text                TEXT,
    entities_json       jsonb,

    media_id            TEXT,
    media_uid           TEXT,
    -- photo, video, sticker, story, video_note, voice, ...
    media_type          TEXT,

    extra_data          jsonb,
    extra_type          TEXT,
    -- RAW_UPDATE_JSON 放入单独的表，避免单表过大
    PRIMARY KEY (chat_id, message_id)
) ;

CREATE TABLE IF NOT EXISTS raw_update
(
    id         bigserial PRIMARY KEY,
    chat_id    bigint,
    message_id bigint,
    raw_update jsonb
);

CREATE INDEX IF NOT EXISTS idx_raw_update_chat_message_id
    ON raw_update (chat_id, message_id);

CREATE TABLE IF NOT EXISTS edit_history
(
    chat_id    bigint NOT NULL,
    message_id bigint NOT NULL,
    edit_id    bigint NOT NULL,
    text       TEXT    NOT NULL,
    PRIMARY KEY (chat_id, message_id, edit_id)
) ;

CREATE OR REPLACE FUNCTION trigger_on_edit_message_fn()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    next_edit_id bigint;
BEGIN
    SELECT e.edit_id + 1
    INTO next_edit_id
    FROM edit_history AS e
    WHERE e.chat_id = OLD.chat_id
      AND e.message_id = OLD.message_id
    ORDER BY e.edit_id DESC
    LIMIT 1;
    IF NOT FOUND THEN
        next_edit_id := 1;
    END IF;

    INSERT INTO edit_history (chat_id, message_id, edit_id, text)
    VALUES (OLD.chat_id,
            OLD.message_id,
            next_edit_id,
            OLD.text);
    RETURN NEW;
END;
$$;

CREATE TRIGGER trigger_on_edit_message
    AFTER UPDATE ON saved_msgs
    FOR EACH ROW
    EXECUTE FUNCTION trigger_on_edit_message_fn();
