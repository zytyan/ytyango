-- encoding: utf-8

-- name: getPicByRateAndRandKey :one
SELECT *
FROM saved_pics
WHERE user_rate = ?
  AND rand_key > ?
LIMIT 1;

-- name: getPicByRateFirst :one
SELECT *
FROM saved_pics
WHERE user_rate = ?
ORDER BY rand_key
LIMIT 1;

-- name: insertPic :one
INSERT INTO saved_pics (file_uid, file_id, bot_rate, rand_key, user_rate)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(file_uid) DO UPDATE SET file_id   = excluded.file_id,
                                    bot_rate  = excluded.bot_rate,
                                    user_rate =
                                        CASE
                                            WHEN excluded.rate_user_count = 0
                                                THEN excluded.bot_rate
                                            ELSE
                                                excluded.user_rate
                                            END
RETURNING *;


-- name: getPicRateCounts :many
SELECT *
FROM pic_rate_counter
ORDER BY rate;

-- name: ratePic :exec
INSERT INTO saved_pics_rating (file_uid, user_id, rating)
VALUES (?, ?, ?);

-- name: getPicRateByUserId :one
SELECT rating
FROM saved_pics_rating
WHERE file_uid = ?
  AND user_id = ?;

-- name: GetNsfwPicByFileUid :one
SELECT *
FROM saved_pics
WHERE file_uid = ?;