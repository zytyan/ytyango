-- name: GetReactions
SELECT reactions_enabled FROM ai_user_settings WHERE user_id=?;

-- name: SetReactions
INSERT INTO ai_user_settings(user_id,reactions_enabled,updated_at) VALUES(?,?,?)
ON CONFLICT(user_id) DO UPDATE SET reactions_enabled=excluded.reactions_enabled,updated_at=excluded.updated_at;

-- name: IsAiResponse
SELECT id FROM ai_runs WHERE status='delivered' AND response_chat_id=? AND response_msg_id=?;

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
