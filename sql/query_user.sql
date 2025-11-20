-- encoding: utf-8

-- name: updateUserBase :one
INSERT INTO users (updated_at, user_id, first_name, last_name, time_zone)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT DO UPDATE SET updated_at=excluded.updated_at,
                          first_name=excluded.first_name,
                          last_name =excluded.last_name,
                          time_zone=excluded.time_zone
RETURNING id;

-- name: updateUserProfilePhoto :exec
INSERT INTO users (user_id, profile_update_at, profile_photo)
VALUES (?, ?, ?)
ON CONFLICT DO UPDATE SET profile_update_at = excluded.profile_update_at,
                          profile_photo     = excluded.profile_photo;



-- name: SetPrprCache :exec
INSERT INTO prpr_caches (profile_photo_uid, prpr_file_id)
VALUES (?, ?);

-- name: GetPrprCache :one
SELECT prpr_file_id
FROM prpr_caches
WHERE profile_photo_uid = ?;
