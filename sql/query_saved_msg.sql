-- encoding: utf-8

-- name: UpsertSavedMsg :one
INSERT INTO saved_msgs (mongo_id, peer_id, from_id, msg_id, date, message, image_text, qr_result)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT DO UPDATE SET mongo_id   = excluded.mongo_id,
                          peer_id    = excluded.peer_id,
                          from_id    = excluded.from_id,
                          msg_id     = excluded.msg_id,
                          date       = excluded.date,
                          message    = excluded.message,
                          image_text = excluded.image_text,
                          qr_result  = excluded.qr_result
RETURNING *;

-- name: ListSavedMsgsAfterId :many
SELECT *
FROM saved_msgs
WHERE id > ?
ORDER BY id ASC
LIMIT ?;

-- name: InsertSavedMsgFts :exec
INSERT INTO saved_msgs_fts (doc_id, message, image_text, qr_result)
VALUES (?, ?, ?, ?);

-- name: DeleteSavedMsgFtsByDocId :exec
DELETE
FROM saved_msgs_fts
WHERE doc_id = ?;

-- name: GetSavedMsgsFtsCursor :one
SELECT last_saved_msg_id
FROM saved_msgs_fts_state
WHERE id = 1;

-- name: UpdateSavedMsgsFtsCursor :exec
INSERT INTO saved_msgs_fts_state (id, last_saved_msg_id)
VALUES (1, ?)
ON CONFLICT(id) DO UPDATE SET last_saved_msg_id = excluded.last_saved_msg_id;

-- name: SearchSavedMsgs :many
SELECT sm.*
FROM saved_msgs sm
         JOIN saved_msgs_fts ON saved_msgs_fts.doc_id = sm.id
WHERE saved_msgs_fts MATCH ?1
  AND sm.peer_id = ?2
  AND (?3 IS NULL OR sm.from_id = ?3)
ORDER BY sm.date DESC, sm.id DESC
LIMIT ?4 OFFSET ?5;
