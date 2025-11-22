-- encoding: utf-8

-- name: getUserById :one
SELECT *
FROM users
WHERE user_id = ?;

-- name: createNewUser :one
INSERT INTO users (updated_at, user_id, first_name, last_name, profile_update_at, profile_photo, timezone)
VALUES (?1, ?2, ?3, ?4, ?1, ?5, ?6)
RETURNING id;

-- name: updateUserBase :one
UPDATE users
SET updated_at=?,
    first_name=?,
    last_name =?
WHERE user_id = ?
RETURNING id;

-- name: updateUserProfilePhoto :exec
UPDATE users
SET profile_update_at = ?,
    profile_photo     = ?
WHERE user_id = ?;

-- name: updateUserTimeZone :exec
INSERT INTO users (user_id, updated_at, timezone)
VALUES (?, ?, ?)
ON CONFLICT DO UPDATE SET updated_at=excluded.updated_at,
                          timezone=excluded.timezone
RETURNING id;

-- name: SetPrprCache :exec
INSERT INTO prpr_caches (profile_photo_uid, prpr_file_id)
VALUES (?, ?);

-- name: GetPrprCache :one
SELECT prpr_file_id
FROM prpr_caches
WHERE profile_photo_uid = ?;
