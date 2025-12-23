-- encoding: utf-8

-- name: GetCocCharAttr :one
SELECT attr_value
FROM character_attrs
WHERE user_id = $1
  AND attr_name = $2;

-- name: GetCocCharAllAttr :many
SELECT attr_name, attr_value
FROM character_attrs
WHERE user_id = $1;

-- name: SetCocCharAttr :exec
INSERT INTO character_attrs
    (user_id, attr_name, attr_value)
VALUES ($1, $2, $3)
ON CONFLICT DO UPDATE SET attr_value=excluded.attr_value;

-- name: DelCocCharAttr :exec
DELETE
FROM character_attrs
WHERE user_id = $1
  AND attr_name = $2;
