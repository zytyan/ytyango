-- encoding: utf-8

-- name: InsertSavedMessage :exec
INSERT INTO saved_msgs (message_id, chat_id, from_user_id, sender_chat_id, date, forward_origin_name, forward_origin_id,
                        message_thread_id, reply_to_message_id, reply_to_chat_id, via_bot_id, edit_date, media_group_id,
                        text, entities_json, media_id, media_uid, media_type, extra_data, extra_type)
VALUES (?, ?, ?, ?, ?,
        ?, ?, ?, ?, ?,
        ?, ?, ?, ?, ?,
        ?, ?, ?, ?, ?)
ON CONFLICT(chat_id, message_id) DO NOTHING;


-- name: GetSavedMessageById :one
SELECT message_id,
       chat_id,
       from_user_id,
       sender_chat_id,
       date,
       forward_origin_name,
       forward_origin_id,
       message_thread_id,
       reply_to_message_id,
       reply_to_chat_id,
       via_bot_id,
       edit_date,
       media_group_id,
       text,
       entities_json,
       media_id,
       media_uid,
       media_type,
       extra_data,
       extra_type
FROM saved_msgs
WHERE chat_id = ?
  AND message_id = ?;

-- name: UpdateMessageText :exec
UPDATE saved_msgs
SET text=?,
    entities_json=?,
    edit_date=?
WHERE chat_id = ?
  AND message_id = ?;

-- name: InsertRawUpdate :exec
INSERT INTO raw_update (id, chat_id, message_id, raw_update)
VALUES (?, ?, ?, ?)
ON CONFLICT(id) DO NOTHING;

-- name: ListEditHistoryByMessage :many
SELECT chat_id,
       message_id,
       edit_id,
       text
FROM edit_history
WHERE chat_id = ?
  AND message_id = ?
ORDER BY edit_id;
