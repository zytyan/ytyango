-- encoding: utf-8

CREATE TABLE saved_pics
(
    file_uid        TEXT    NOT NULL,
    file_id         TEXT    NOT NULL, -- 插入时，若 file_uid 相同，则更新 file_id
    bot_rate        INTEGER NOT NULL, -- 目前为 [-1,7] 的整数，-1 时相当于删除
    rand_key        bigint  NOT NULL,
    -- user_rate 为生成列：有评分时为平均分；否则回退到 bot_rate
    user_rate       INTEGER GENERATED ALWAYS AS (
        CASE
            WHEN rate_user_count > 0
                THEN ROUND(user_rating_sum::numeric / rate_user_count)::int
            ELSE bot_rate
        END
    ) STORED,
    user_rating_sum bigint NOT NULL DEFAULT 0,
    rate_user_count bigint NOT NULL DEFAULT 0,
    PRIMARY KEY (file_uid),
    UNIQUE (user_rate, rand_key),
    UNIQUE (rand_key)                 -- 确保 rand_key 自身唯一
) ;


CREATE TABLE IF NOT EXISTS saved_pics_rating
(
    file_uid TEXT    NOT NULL,
    user_id  bigint NOT NULL,
    rating   INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (file_uid, user_id),
    FOREIGN KEY (file_uid) REFERENCES saved_pics (file_uid)
) ;

CREATE OR REPLACE FUNCTION saved_pics_rating_insert_trigger_fn()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    UPDATE saved_pics
    SET user_rating_sum = user_rating_sum + NEW.rating,
        rate_user_count = rate_user_count + 1
    WHERE file_uid = NEW.file_uid;
    RETURN NEW;
END;
$$;

CREATE OR REPLACE FUNCTION saved_pics_rating_update_trigger_fn()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    UPDATE saved_pics
    SET user_rating_sum = user_rating_sum - OLD.rating + NEW.rating
    WHERE file_uid = OLD.file_uid;
    RETURN NEW;
END;
$$;

CREATE OR REPLACE FUNCTION saved_pics_rating_delete_trigger_fn()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    UPDATE saved_pics
    SET user_rating_sum = user_rating_sum - OLD.rating,
        rate_user_count = rate_user_count - 1
    WHERE file_uid = OLD.file_uid;
    RETURN OLD;
END;
$$;

CREATE TRIGGER saved_pics_rating_insert_trigger
    AFTER INSERT ON saved_pics_rating
    FOR EACH ROW
    EXECUTE FUNCTION saved_pics_rating_insert_trigger_fn();

CREATE TRIGGER saved_pics_rating_update_trigger
    AFTER UPDATE ON saved_pics_rating
    FOR EACH ROW
    EXECUTE FUNCTION saved_pics_rating_update_trigger_fn();

CREATE TRIGGER saved_pics_rating_delete_trigger
    AFTER DELETE ON saved_pics_rating
    FOR EACH ROW
    EXECUTE FUNCTION saved_pics_rating_delete_trigger_fn();


CREATE TABLE IF NOT EXISTS pic_rate_counter
(
    rate  INTEGER NOT NULL,
    count INTEGER NOT NULL,
    PRIMARY KEY (rate)
) ;


CREATE OR REPLACE FUNCTION saved_pics_update_trigger_fn()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    UPDATE pic_rate_counter
    SET count = count + 1
    WHERE rate = NEW.user_rate;

    UPDATE pic_rate_counter
    SET count = count - 1
    WHERE rate = OLD.user_rate;
    RETURN NEW;
END;
$$;

CREATE OR REPLACE FUNCTION saved_pics_insert_trigger_fn()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    INSERT INTO pic_rate_counter (rate, count)
    VALUES (NEW.user_rate, 1)
    ON CONFLICT (rate) DO UPDATE SET count = pic_rate_counter.count + 1;
    RETURN NEW;
END;
$$;

CREATE OR REPLACE FUNCTION saved_pics_delete_trigger_fn()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    UPDATE pic_rate_counter
    SET count = count - 1
    WHERE rate = OLD.user_rate;
    RETURN OLD;
END;
$$;

CREATE TRIGGER saved_pics_update_trigger
    AFTER UPDATE ON saved_pics
    FOR EACH ROW
    EXECUTE FUNCTION saved_pics_update_trigger_fn();


CREATE TRIGGER saved_pics_insert_trigger
    AFTER INSERT ON saved_pics
    FOR EACH ROW
    EXECUTE FUNCTION saved_pics_insert_trigger_fn();


CREATE TRIGGER saved_pics_delete_trigger
    AFTER DELETE ON saved_pics
    FOR EACH ROW
    EXECUTE FUNCTION saved_pics_delete_trigger_fn();
