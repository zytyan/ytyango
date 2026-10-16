-- Incoming-message deduplication runs before reaction and must not scan payload-bearing rows.
CREATE INDEX idx_ai_runs_request ON ai_runs(request_chat_id,request_msg_id,id DESC);
