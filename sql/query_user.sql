-- encoding: utf-8

-- name: getUserById :one
SELECT *
FROM users
WHERE user_id = ?;

-- name: createNewUser :one
INSERT INTO users (updated_at, user_id, first_name, last_name, username, profile_update_at, profile_photo, timezone)
VALUES (?1, ?2, ?3, ?4, ?5, ?1, ?6, ?7)
RETURNING id;

-- name: updateUserBase :one
UPDATE users
SET updated_at=?2,
    first_name=?3,
    last_name =?4,
    username  = ?5
WHERE user_id = ?1
RETURNING id;

-- name: updateUserProfilePhoto :exec
UPDATE users
SET profile_update_at = ?2,
    profile_photo     = ?3
WHERE user_id = ?1;

-- name: updateUserTimeZone :exec
UPDATE users
SET timezone = ?2
WHERE user_id = ?1
RETURNING id;

-- name: SetPrprCache :exec
INSERT INTO prpr_caches (profile_photo_uid, prpr_file_id)
VALUES (?, ?);

-- name: GetPrprCache :one
SELECT prpr_file_id
FROM prpr_caches
WHERE profile_photo_uid = ?;
