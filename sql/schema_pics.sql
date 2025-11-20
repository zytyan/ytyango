-- encoding: utf-8

CREATE TABLE IF NOT EXISTS saved_pics
(
    file_uid        TEXT    NOT NULL,
    file_id         TEXT    NOT NULL, --插入时，若file_uid相同，则更新file_id
    bot_rate        INTEGER NOT NULL, -- 目前为[-1,7]的整数，-1时相当于删除
    rand_key        INTEGER NOT NULL,
    user_rate       INTEGER NOT NULL, -- 用户的评分，默认是bot的评分
    user_rating_sum INTEGER NOT NULL DEFAULT 0,
    rate_user_count INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (file_uid),
    UNIQUE (user_rate, rand_key),
    UNIQUE (rand_key)                 -- 再加一个rand_key自身的索引，确保user_rate变动时不会非常不巧碰上另一个unique
) WITHOUT ROWID, STRICT;


CREATE TABLE IF NOT EXISTS saved_pics_rating
(
    file_uid TEXT    NOT NULL,
    user_id  INTEGER NOT NULL,
    rating   INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (file_uid, user_id),
    FOREIGN KEY (file_uid) REFERENCES saved_pics (file_uid)
) WITHOUT ROWID, STRICT;


CREATE TRIGGER IF NOT EXISTS saved_pics_rating_insert_trigger
    AFTER INSERT
    ON saved_pics_rating
BEGIN
    UPDATE saved_pics
    SET user_rating_sum = user_rating_sum + new.rating,
        rate_user_count = rate_user_count + 1,
        user_rate       = CASE
                              WHEN rate_user_count + 1 > 0
                                  THEN CAST(ROUND((user_rating_sum + new.rating) * 1.0 / (rate_user_count + 1)) AS INTEGER)
                              ELSE user_rate
            END
    WHERE file_uid = new.file_uid;
END;

CREATE TRIGGER IF NOT EXISTS saved_pics_rating_update_trigger
    AFTER UPDATE
    ON saved_pics_rating
BEGIN
    UPDATE saved_pics
    SET user_rating_sum = user_rating_sum - old.rating + new.rating,
        user_rate       = CASE
                              WHEN rate_user_count > 0
                                  THEN CAST(ROUND((user_rating_sum - old.rating + new.rating) * 1.0 / rate_user_count) AS INTEGER)
                              ELSE user_rate
            END
    WHERE file_uid = old.file_uid;
END;

CREATE TRIGGER IF NOT EXISTS saved_pics_rating_delete_trigger
    AFTER DELETE
    ON saved_pics_rating
BEGIN
    UPDATE saved_pics
    SET user_rating_sum = user_rating_sum - old.rating,
        rate_user_count = rate_user_count - 1,
        user_rate       = CASE
                              WHEN rate_user_count - 1 > 0
                                  THEN CAST(ROUND((user_rating_sum - old.rating) * 1.0 / (rate_user_count - 1)) AS INTEGER)
                              ELSE bot_rate -- 用户评分清空后回到 bot_rate
            END
    WHERE file_uid = old.file_uid;
END;


CREATE TABLE IF NOT EXISTS pic_rate_counter
(
    rate  INTEGER NOT NULL,
    count INTEGER NOT NULL,
    PRIMARY KEY (rate)
) WITHOUT ROWID, STRICT;


CREATE TRIGGER IF NOT EXISTS saved_pics_update_trigger
    AFTER UPDATE
    ON saved_pics
BEGIN
    UPDATE pic_rate_counter
    SET count = count + 1
    WHERE rate = new.user_rate;
    UPDATE pic_rate_counter
    SET count = count - 1
    WHERE rate = old.user_rate;
END;


CREATE TRIGGER IF NOT EXISTS saved_pics_insert_trigger
    AFTER INSERT
    ON saved_pics
BEGIN
    -- 新图片插入时，增加其 user_rate 对应的计数
    INSERT INTO pic_rate_counter (rate, count)
    VALUES (new.user_rate, 1)
    ON CONFLICT (rate) DO UPDATE SET count = count + 1;
END;


CREATE TRIGGER IF NOT EXISTS saved_pics_delete_trigger
    AFTER DELETE
    ON saved_pics
BEGIN
    -- 旧图片删除时，减少其 user_rate 对应的计数
    UPDATE pic_rate_counter
    SET count = count - 1
    WHERE rate = old.user_rate;
END;
