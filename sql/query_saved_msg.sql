-- encoding: utf-8

-- name: CreateNewMessage :exec
INSERT INTO saved_msgs (message_id, chat_id, from_user_id, sender_chat_id, date, forward_origin_name, forward_origin_id,
                        message_thread_id, reply_to_message_id, reply_to_chat_id, via_bot_id, edit_date, media_group_id,
                        text, entities_json, media_id, media_uid, media_type, extra_data, extra_type)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);


-- name: GetSavedMessageById :one
SELECT *
FROM saved_msgs
WHERE chat_id = ?
  AND message_id = ?;

-- name: UpdateMessageText :exec
UPDATE saved_msgs
SET text=?
WHERE chat_id = ?
  AND message_id = ?