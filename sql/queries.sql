-- name: GetReactions
SELECT reactions_enabled FROM ai_user_settings WHERE user_id=?;

-- name: SetReactions
INSERT INTO ai_user_settings(user_id,reactions_enabled,updated_at) VALUES(?,?,?)
ON CONFLICT(user_id) DO UPDATE SET reactions_enabled=excluded.reactions_enabled,updated_at=excluded.updated_at;

-- name: IsAiResponse
SELECT r.id FROM ai_runs r JOIN ai_sessions s ON s.id=r.session_id
WHERE r.status='delivered' AND r.response_chat_id=? AND r.response_msg_id=? AND COALESCE(s.topic_id,0)=?;

-- name: SaveFact
INSERT INTO traditional_facts(feature,chat_id,topic_id,request_msg_id,reply_msg_id,status,context_text,created_at,delivery_status)
VALUES(?,?,?,?,?,?,?,?,?);

-- name: ReplyFacts
SELECT feature,request_msg_id,reply_msg_id,status,context_text,created_at FROM traditional_facts
WHERE chat_id=? AND topic_id=? AND reply_msg_id=? AND delivery_status='delivered' ORDER BY id;

-- name: FreezeRunContext
INSERT INTO ai_run_context(run_id,snapshot) VALUES(?,?) ON CONFLICT(run_id) DO NOTHING;

-- name: GetRunContext
SELECT snapshot FROM ai_run_context WHERE run_id=?;

-- name: RequestFacts
SELECT feature,request_msg_id,reply_msg_id,status,context_text,created_at FROM traditional_facts
WHERE chat_id=? AND topic_id=? AND request_msg_id=? ORDER BY id;

-- name: RunForRequest
SELECT * FROM ai_runs WHERE request_chat_id=? AND request_msg_id=? ORDER BY id DESC LIMIT 1;

-- name: GetPrprCache
SELECT prpr_file_id FROM prpr_caches WHERE profile_photo_uid=?;

-- name: SetPrprCache
INSERT INTO prpr_caches(profile_photo_uid,prpr_file_id) VALUES(?,?)
ON CONFLICT(profile_photo_uid) DO UPDATE SET prpr_file_id=excluded.prpr_file_id;

-- name: EnsureChatConfig
INSERT INTO chat_cfg(id,auto_cvt_bili,auto_calculate,auto_exchange,auto_check_adult,enable_coc,resp_nsfw_msg,timezone)
VALUES(?,0,0,0,0,0,0,?) ON CONFLICT(id) DO NOTHING;

-- name: SetChatConfig
UPDATE chat_cfg SET auto_cvt_bili=?,auto_calculate=?,auto_exchange=?,auto_check_adult=?,resp_nsfw_msg=? WHERE id=?;
