-- encoding: utf-8

CREATE TABLE IF NOT EXISTS saved_msgs
(
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    mongo_id   TEXT         NOT NULL,
    peer_id    INTEGER      NOT NULL,
    from_id    INTEGER      NOT NULL,
    msg_id     INTEGER      NOT NULL,
    date       INT_UNIX_SEC NOT NULL,
    message    TEXT,
    image_text TEXT,
    qr_result  TEXT,
    UNIQUE (mongo_id),
    UNIQUE (peer_id, msg_id)
);

CREATE INDEX IF NOT EXISTS idx_saved_msgs_peer_date
    ON saved_msgs (peer_id, date DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_saved_msgs_from
    ON saved_msgs (from_id, date DESC);

CREATE VIRTUAL TABLE IF NOT EXISTS saved_msgs_fts
    USING fts5(doc_id UNINDEXED, message, image_text, qr_result, tokenize = 'unicode61');

CREATE TABLE IF NOT EXISTS saved_msgs_fts_state
(
    id                INTEGER PRIMARY KEY CHECK (id = 1),
    last_saved_msg_id INTEGER NOT NULL DEFAULT 0
) WITHOUT ROWID;

INSERT INTO saved_msgs_fts_state (id, last_saved_msg_id)
VALUES (1, 0)
ON CONFLICT DO NOTHING;
