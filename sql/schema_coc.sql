-- encoding: utf-8

CREATE TABLE IF NOT EXISTS character_attrs
(
    user_id    bigint NOT NULL,
    attr_name  TEXT    NOT NULL,
    attr_value TEXT    NOT NULL,
    PRIMARY KEY (user_id, attr_name)
);
