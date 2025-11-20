-- encoding: utf-8

CREATE TABLE IF NOT EXISTS users
(
    id                INTEGER PRIMARY KEY AUTOINCREMENT,
    updated_at        INT_UNIX_SEC NOT NULL,
    user_id           INTEGER      NOT NULL UNIQUE,
    first_name        TEXT         NOT NULL,
    last_name         TEXT,
    profile_update_at INT_UNIX_SEC NOT NULL,
    profile_photo     TEXT,
    time_zone         INTEGER DEFAULT 480 -- 8:00，+8小时
);

CREATE TABLE IF NOT EXISTS prpr_caches
(
    profile_photo_uid TEXT NOT NULL PRIMARY KEY,
    prpr_file_id      TEXT NOT NULL
) WITHOUT ROWID;