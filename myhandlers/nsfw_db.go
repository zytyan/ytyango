package myhandlers

import (
	"errors"
	"fmt"
	"main/globalcfg"
	"math/rand"

	"github.com/mattn/go-sqlite3"
	"gorm.io/gorm"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}
func initNsfwPicDb() {
	tx := globalcfg.GetDb().Begin()

	tx.Exec(`
CREATE TABLE IF NOT EXISTS saved_pics
(
    file_uid        TEXT    NOT NULL,
    file_id         TEXT    NOT NULL,   --插入时，若file_uid相同，则更新file_id
    bot_rate        INTEGER NOT NULL,   -- 目前为[-1,7]的整数，-1时相当于删除
    rand_key        INTEGER NOT NULL,
    user_rate       INTEGER NOT NULL, -- 用户的评分，默认是bot的评分
    user_rating_sum INTEGER NOT NULL DEFAULT 0,
    rate_user_count INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (file_uid),
    UNIQUE (user_rate, rand_key),
    UNIQUE (rand_key) -- 再加一个rand_key自身的索引，确保user_rate变动时不会非常不巧碰上另一个unique
) WITHOUT ROWID , STRICT;
`)
	tx.Exec(`
CREATE TABLE IF NOT EXISTS saved_pics_rating
(
    file_uid TEXT    NOT NULL,
    user_id  INTEGER NOT NULL,
    rating   INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (file_uid, user_id),
    FOREIGN KEY (file_uid) REFERENCES saved_pics (file_uid)
) WITHOUT ROWID , STRICT;
`)
	tx.Exec(`
CREATE TRIGGER IF NOT EXISTS saved_pics_rating_insert_trigger
AFTER INSERT ON saved_pics_rating
BEGIN
    UPDATE saved_pics
    SET user_rating_sum = user_rating_sum + new.rating,
        rate_user_count = rate_user_count + 1,
        user_rate = CASE 
            WHEN rate_user_count + 1 > 0 THEN CAST(ROUND((user_rating_sum + new.rating) * 1.0 / (rate_user_count + 1)) AS INTEGER)
            ELSE user_rate
        END
    WHERE file_uid = new.file_uid;
END;`)
	tx.Exec(`
CREATE TRIGGER IF NOT EXISTS saved_pics_rating_update_trigger
AFTER UPDATE ON saved_pics_rating
BEGIN
    UPDATE saved_pics
    SET user_rating_sum = user_rating_sum - old.rating + new.rating,
        user_rate = CASE 
            WHEN rate_user_count > 0 THEN CAST(ROUND((user_rating_sum - old.rating + new.rating) * 1.0 / rate_user_count) AS INTEGER)
            ELSE user_rate
        END
    WHERE file_uid = old.file_uid;
END;`)
	tx.Exec(`
CREATE TRIGGER IF NOT EXISTS saved_pics_rating_delete_trigger
AFTER DELETE ON saved_pics_rating
BEGIN
    UPDATE saved_pics
    SET user_rating_sum = user_rating_sum - old.rating,
        rate_user_count = rate_user_count - 1,
        user_rate = CASE
            WHEN rate_user_count - 1 > 0 THEN CAST(ROUND((user_rating_sum - old.rating) * 1.0 / (rate_user_count - 1)) AS INTEGER)
            ELSE bot_rate -- 用户评分清空后回到 bot_rate
        END
    WHERE file_uid = old.file_uid;
END;
`)
	check(tx.Error)
	tx.Commit()
}
func init() {
	initNsfwPicDb()
}

func getRandomPicByRate(rate int) string {
	rnd := int64(rand.Uint64())
	stmt1 := `SELECT file_id
    FROM saved_pics
    WHERE user_rate = ? AND rand_key > ? 
    LIMIT 1`
	stmt2 := `SELECT file_id 
			FROM saved_pics 
			WHERE user_rate = ? 
			ORDER BY rand_key LIMIT 1`
	tx := globalcfg.GetDb()
	tx.Raw(stmt1, rnd, rate)
	if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		tx.Raw(stmt2, rate)
		if tx.Error != nil {
			log.Errorf("can't find random pic by rate %d", rate)
			return ""
		}
	} else if tx.Error != nil {
		log.Errorf("fetch data from database error: %s, rate: %d", tx.Error, rate)
		return ""
	}
	var result string
	row := tx.Row()
	if row == nil {
		log.Errorf("can't find random pic by rate %d, row is nil", rate)
		return ""
	}
	err := row.Scan(&result)
	if err != nil {
		log.Errorf("scan data from database error: %s, rate: %d", err, rate)
		return ""
	}
	return result
}

func addPicToDb(fileUid, fileId string, botRate int) error {

	stmt := `INSERT INTO saved_pics (file_uid, file_id, bot_rate, rand_key, user_rate) VALUES (?, ?, ?, ?, ?) 
			 ON CONFLICT(file_uid) DO UPDATE SET
	             file_id = excluded.file_id,
	             bot_rate = excluded.bot_rate`
	for i := 0; i < 3; i++ {
		rnd := int64(rand.Uint64())
		tx := globalcfg.GetDb().Exec(stmt, fileUid, fileId, botRate, rnd, botRate)
		if tx.Error != nil {
			var err sqlite3.Error
			if errors.As(tx.Error, &err) && errors.Is(err.Code, sqlite3.ErrConstraint) {
				continue
			}
			return tx.Error
		}
		tx.Commit()
		return nil
	}
	return fmt.Errorf("failed after 16 retries (rand_key conflicts)")
}
func getRandomNsfwAdult() string {
	return getRandomPicByRate(6)
}

func getRandomNsfwRacy() string {
	if rand.Int()%2 == 0 {
		return getRandomPicByRate(4)
	} else {
		return getRandomPicByRate(2)
	}
}
