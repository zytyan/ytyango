-- encoding: utf-8

-- name: getNsfwPicByRateAndRandKey :one
SELECT *
FROM saved_pics
WHERE user_rate = ?
  AND rand_key > ?
LIMIT 1;

-- name: getNsfwPicByRateFirst :one
SELECT *
FROM saved_pics
WHERE user_rate = ?
ORDER BY rand_key
LIMIT 1;

-- name: createOrUpdateNsfwPic :one
INSERT INTO saved_pics (file_uid, file_id, bot_rate, rand_key)
VALUES (?, ?, ?, ?)
ON CONFLICT(file_uid) DO UPDATE SET file_id   = excluded.file_id,
                                    bot_rate  = excluded.bot_rate
RETURNING *;


-- name: listNsfwPicRateCounter :many
SELECT *
FROM pic_rate_counter
ORDER BY rate;

-- name: createNsfwPicUserRate :exec
INSERT INTO saved_pics_rating (file_uid, user_id, rating)
VALUES (?, ?, ?);

-- name: updateNsfwPicUserRate :exec
UPDATE saved_pics_rating
SET rating=?
WHERE file_uid = ?
  AND user_id = ?;

-- name: getNsfwPicRateByUserId :one
SELECT rating
FROM saved_pics_rating
WHERE file_uid = ?
  AND user_id = ?;

-- name: GetNsfwPicByFileUid :one
SELECT *
FROM saved_pics
WHERE file_uid = ?;

-- name: ListNsfwPicUserRatesByFileUid :many
SELECT rating, COUNT(*)
FROM saved_pics_rating
WHERE file_uid = ?
GROUP BY rating;