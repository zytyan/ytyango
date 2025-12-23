-- encoding: utf-8

-- name: InsertSavedMessage :exec
INSERT INTO saved_msgs (message_id, chat_id, from_user_id, sender_chat_id, date, forward_origin_name, forward_origin_id,
                        message_thread_id, reply_to_message_id, reply_to_chat_id, via_bot_id, edit_date, media_group_id,
                        text, entities_json, media_id, media_uid, media_type, extra_data, extra_type)
VALUES ($1, $2, $3, $4, $5,
        $6, $7, $8, $9, $10,
        $11, $12, $13, $14, $15,
        $16, $17, $18, $19, $20)
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
WHERE chat_id = $1
  AND message_id = $2;

-- name: UpdateMessageText :exec
UPDATE saved_msgs
SET text=$1,
    entities_json=$2,
    edit_date=$3
WHERE chat_id = $4
  AND message_id = $5;

-- name: InsertRawUpdate :exec
INSERT INTO raw_update (chat_id, message_id, raw_update)
VALUES ($1, $2, $3);

-- name: ListEditHistoryByMessage :many
SELECT chat_id,
       message_id,
       edit_id,
       text
FROM edit_history
WHERE chat_id = $1
  AND message_id = $2
ORDER BY edit_id;
