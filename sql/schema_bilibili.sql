-- encoding: utf-8
CREATE TABLE IF NOT EXISTS bili_inline_results
(
    uid     bigserial PRIMARY KEY,
    text    TEXT      NOT NULL DEFAULT '',
    chat_id bigint    NOT NULL DEFAULT 0,
    msg_id  bigint    NOT NULL DEFAULT 0
);
