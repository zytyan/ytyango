-- encoding: utf-8

CREATE TABLE IF NOT EXISTS users
(
    id                bigserial PRIMARY KEY,
    updated_at        timestamptz  NOT NULL,
    user_id           bigint       NOT NULL UNIQUE,
    first_name        TEXT         NOT NULL,
    last_name         TEXT,
    profile_update_at timestamptz  NOT NULL,
    profile_photo     TEXT,
    timezone          INTEGER NOT NULL DEFAULT 480 -- 8:00，+8小时
);

CREATE TABLE IF NOT EXISTS prpr_caches
(
    profile_photo_uid TEXT NOT NULL PRIMARY KEY,
    prpr_file_id      TEXT NOT NULL
);
