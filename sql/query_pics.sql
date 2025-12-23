-- encoding: utf-8

-- name: getNsfwPicByRateAndRandKey :one
SELECT *
FROM saved_pics
WHERE user_rate = $1
  AND rand_key > $2
LIMIT 1;

-- name: getNsfwPicByRateFirst :one
SELECT *
FROM saved_pics
WHERE user_rate = $1
ORDER BY rand_key
LIMIT 1;

-- name: createOrUpdateNsfwPic :one
INSERT INTO saved_pics (file_uid, file_id, bot_rate, rand_key)
VALUES ($1, $2, $3, $4)
ON CONFLICT(file_uid) DO UPDATE SET file_id   = excluded.file_id,
                                    bot_rate  = excluded.bot_rate
RETURNING *;


-- name: listNsfwPicRateCounter :many
SELECT *
FROM pic_rate_counter
ORDER BY rate;

-- name: createNsfwPicUserRate :exec
INSERT INTO saved_pics_rating (file_uid, user_id, rating)
VALUES ($1, $2, $3);

-- name: updateNsfwPicUserRate :exec
UPDATE saved_pics_rating
SET rating=$1
WHERE file_uid = $2
  AND user_id = $3;

-- name: getNsfwPicRateByUserId :one
SELECT rating
FROM saved_pics_rating
WHERE file_uid = $1
  AND user_id = $2;

-- name: GetNsfwPicByFileUid :one
SELECT *
FROM saved_pics
WHERE file_uid = $1;

-- name: ListNsfwPicUserRatesByFileUid :many
SELECT rating, COUNT(*)
FROM saved_pics_rating
WHERE file_uid = $1
GROUP BY rating;
