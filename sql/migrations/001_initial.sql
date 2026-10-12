CREATE TABLE IF NOT EXISTS users (
  user_id INTEGER NOT NULL PRIMARY KEY,
  updated_at INT_UNIX_SEC NOT NULL,
  first_name TEXT NOT NULL,
  last_name TEXT,
  username TEXT
) WITHOUT ROWID;
CREATE TABLE IF NOT EXISTS prpr_caches (
  profile_photo_uid TEXT NOT NULL PRIMARY KEY,
  prpr_file_id TEXT NOT NULL
) WITHOUT ROWID;

CREATE TABLE IF NOT EXISTS chat_cfg (
  id INTEGER PRIMARY KEY NOT NULL,
  auto_cvt_bili INT_BOOL NOT NULL CHECK (auto_cvt_bili IN (0,1)),
  auto_calculate INT_BOOL NOT NULL CHECK (auto_calculate IN (0,1)),
  auto_exchange INT_BOOL NOT NULL CHECK (auto_exchange IN (0,1)),
  auto_check_adult INT_BOOL NOT NULL CHECK (auto_check_adult IN (0,1)),
  enable_coc INT_BOOL NOT NULL CHECK (enable_coc IN (0,1)),
  resp_nsfw_msg INT_BOOL NOT NULL CHECK (resp_nsfw_msg IN (0,1)),
  timezone INTEGER NOT NULL CHECK (timezone < 86400 AND timezone > -86400)
);
CREATE TABLE chat_stat_daily (
  chat_id INTEGER NOT NULL,
  stat_date INTEGER NOT NULL,
  message_count INTEGER NOT NULL DEFAULT 0,
  photo_count INTEGER NOT NULL DEFAULT 0,
  video_count INTEGER NOT NULL DEFAULT 0,
  sticker_count INTEGER NOT NULL DEFAULT 0,
  forward_count INTEGER NOT NULL DEFAULT 0,
  mars_count INTEGER NOT NULL DEFAULT 0,
  max_mars_count INTEGER NOT NULL DEFAULT 0,
  racy_count INTEGER NOT NULL DEFAULT 0,
  adult_count INTEGER NOT NULL DEFAULT 0,
  download_video_count INTEGER NOT NULL DEFAULT 0,
  download_audio_count INTEGER NOT NULL DEFAULT 0,
  dio_add_user_count INTEGER NOT NULL DEFAULT 0,
  dio_ban_user_count INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (chat_id, stat_date)
) WITHOUT ROWID;
CREATE TABLE chat_stat_user_daily (
  chat_id INTEGER NOT NULL,
  stat_date INTEGER NOT NULL,
  user_id INTEGER NOT NULL,
  message_count INTEGER NOT NULL CHECK (message_count >= 0),
  message_length INTEGER NOT NULL CHECK (message_length >= 0),
  PRIMARY KEY (chat_id, stat_date, user_id),
  FOREIGN KEY (chat_id, stat_date) REFERENCES chat_stat_daily(chat_id, stat_date) ON DELETE CASCADE
) WITHOUT ROWID, STRICT;
CREATE TABLE chat_stat_bucket_daily (
  chat_id INTEGER NOT NULL,
  stat_date INTEGER NOT NULL,
  bucket INTEGER NOT NULL CHECK (bucket BETWEEN 0 AND 143),
  message_count INTEGER NOT NULL CHECK (message_count >= 0),
  first_msg_id INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (chat_id, stat_date, bucket),
  FOREIGN KEY (chat_id, stat_date) REFERENCES chat_stat_daily(chat_id, stat_date) ON DELETE CASCADE
) WITHOUT ROWID, STRICT;

CREATE TABLE saved_pics (
  file_uid TEXT NOT NULL,
  file_id TEXT NOT NULL,
  bot_rate INTEGER NOT NULL CHECK (bot_rate BETWEEN -1 AND 7),
  rand_key INTEGER NOT NULL,
  user_rate INTEGER NOT NULL GENERATED ALWAYS AS (
    CASE WHEN rate_user_count > 0
      THEN CAST(ROUND(user_rating_sum * 1.0 / rate_user_count) AS INTEGER)
      ELSE bot_rate END
  ) STORED,
  user_rating_sum INTEGER NOT NULL DEFAULT 0,
  rate_user_count INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (file_uid),
  UNIQUE (user_rate, rand_key),
  UNIQUE (rand_key)
) WITHOUT ROWID, STRICT;
CREATE TABLE saved_pics_rating (
  file_uid TEXT NOT NULL,
  user_id INTEGER NOT NULL,
  rating INTEGER NOT NULL DEFAULT 0 CHECK (rating BETWEEN 0 AND 7),
  PRIMARY KEY (file_uid, user_id),
  FOREIGN KEY (file_uid) REFERENCES saved_pics(file_uid) ON DELETE CASCADE
) WITHOUT ROWID, STRICT;
CREATE TRIGGER saved_pics_rating_insert_trigger AFTER INSERT ON saved_pics_rating BEGIN
  UPDATE saved_pics SET user_rating_sum=user_rating_sum+new.rating,
    rate_user_count=rate_user_count+1 WHERE file_uid=new.file_uid;
END;
CREATE TRIGGER saved_pics_rating_update_trigger AFTER UPDATE ON saved_pics_rating BEGIN
  UPDATE saved_pics SET user_rating_sum=user_rating_sum-old.rating+new.rating WHERE file_uid=old.file_uid;
END;
CREATE TRIGGER saved_pics_rating_delete_trigger AFTER DELETE ON saved_pics_rating BEGIN
  UPDATE saved_pics SET user_rating_sum=user_rating_sum-old.rating,
    rate_user_count=rate_user_count-1 WHERE file_uid=old.file_uid;
END;
CREATE TABLE pic_rate_counter (
  rate INTEGER NOT NULL,
  count INTEGER NOT NULL,
  PRIMARY KEY (rate)
) WITHOUT ROWID, STRICT;
CREATE TRIGGER saved_pics_update_trigger AFTER UPDATE ON saved_pics BEGIN
  UPDATE pic_rate_counter SET count=count+1 WHERE rate=new.user_rate;
  UPDATE pic_rate_counter SET count=count-1 WHERE rate=old.user_rate;
END;
CREATE TRIGGER saved_pics_insert_trigger AFTER INSERT ON saved_pics BEGIN
  INSERT INTO pic_rate_counter(rate,count) VALUES(new.user_rate,1)
    ON CONFLICT(rate) DO UPDATE SET count=count+1;
END;
CREATE TRIGGER saved_pics_delete_trigger AFTER DELETE ON saved_pics BEGIN
  UPDATE pic_rate_counter SET count=count-1 WHERE rate=old.user_rate;
END;

CREATE TABLE IF NOT EXISTS yt_dl_results (
  url TEXT NOT NULL,
  audio_only INT_BOOL NOT NULL CHECK (audio_only IN (0,1)),
  resolution INTEGER NOT NULL,
  file_id TEXT NOT NULL,
  title TEXT NOT NULL,
  description TEXT NOT NULL,
  uploader TEXT NOT NULL,
  upload_count INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (url, audio_only, resolution)
) WITHOUT ROWID;
CREATE INDEX idx_yt_dl_results_file_id ON yt_dl_results(file_id);
CREATE TABLE IF NOT EXISTS bili_inline_results (
  uid INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
  text TEXT NOT NULL DEFAULT '',
  chat_id INTEGER NOT NULL DEFAULT 0,
  msg_id INTEGER NOT NULL DEFAULT 0,
  created_at INT_UNIX_SEC NOT NULL
);
CREATE INDEX idx_bili_inline_results_created_at ON bili_inline_results(created_at);

CREATE TABLE ts_schema_versions (
  version INTEGER PRIMARY KEY,
  name TEXT NOT NULL UNIQUE,
  checksum TEXT NOT NULL,
  applied_at INTEGER NOT NULL
) STRICT;
CREATE TABLE ai_chat_settings (
  chat_id INTEGER PRIMARY KEY,
  default_provider TEXT NOT NULL CHECK (length(default_provider)>0),
  default_model TEXT NOT NULL CHECK (length(default_model)>0),
  show_usage INTEGER NOT NULL DEFAULT 0 CHECK (show_usage IN (0,1)),
  updated_at INTEGER NOT NULL
) WITHOUT ROWID, STRICT;
CREATE TABLE ai_sessions (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  chat_id INTEGER NOT NULL,
  topic_id INTEGER CHECK (topic_id IS NULL OR topic_id>=0),
  chat_name TEXT NOT NULL,
  chat_type TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active','closed','frozen')),
  provider TEXT NOT NULL CHECK (length(provider)>0),
  model TEXT NOT NULL CHECK (length(model)>0),
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL,
  total_input_tokens INTEGER NOT NULL DEFAULT 0 CHECK (total_input_tokens>=0),
  total_output_tokens INTEGER NOT NULL DEFAULT 0 CHECK (total_output_tokens>=0),
  total_cached_input_tokens INTEGER NOT NULL DEFAULT 0 CHECK (total_cached_input_tokens>=0),
  history_rebuild_lossy INTEGER NOT NULL DEFAULT 0 CHECK (history_rebuild_lossy IN (0,1))
) STRICT;
CREATE INDEX idx_ai_sessions_chat_topic_updated ON ai_sessions(chat_id,topic_id,updated_at DESC);
CREATE TABLE ai_messages (
  chat_id INTEGER NOT NULL,
  msg_id INTEGER NOT NULL,
  sent_at INTEGER NOT NULL,
  user_id INTEGER NOT NULL,
  username TEXT NOT NULL,
  atable_username TEXT,
  msg_type TEXT NOT NULL CHECK (length(msg_type)>0),
  text TEXT,
  reply_to_msg_id INTEGER,
  PRIMARY KEY (chat_id,msg_id)
) WITHOUT ROWID, STRICT;
CREATE TABLE media_objects (
  sha256 TEXT PRIMARY KEY CHECK (length(sha256)=64 AND sha256 NOT GLOB '*[^0-9a-f]*'),
  relative_path TEXT NOT NULL UNIQUE,
  byte_size INTEGER NOT NULL CHECK (byte_size>=0),
  mime_type TEXT NOT NULL CHECK (length(mime_type)>0),
  created_at INTEGER NOT NULL
) WITHOUT ROWID, STRICT;
CREATE TABLE ai_message_media (
  chat_id INTEGER NOT NULL,
  msg_id INTEGER NOT NULL,
  ordinal INTEGER NOT NULL CHECK (ordinal>=0),
  media_sha256 TEXT NOT NULL,
  media_kind TEXT NOT NULL CHECK (length(media_kind)>0),
  PRIMARY KEY (chat_id,msg_id,ordinal),
  FOREIGN KEY (chat_id,msg_id) REFERENCES ai_messages(chat_id,msg_id) ON DELETE CASCADE,
  FOREIGN KEY (media_sha256) REFERENCES media_objects(sha256) ON DELETE RESTRICT
) WITHOUT ROWID, STRICT;
CREATE INDEX idx_ai_message_media_hash ON ai_message_media(media_sha256);
CREATE TABLE ai_session_messages (
  session_id INTEGER NOT NULL,
  position INTEGER NOT NULL CHECK (position>=0),
  chat_id INTEGER NOT NULL,
  msg_id INTEGER NOT NULL,
  role TEXT NOT NULL CHECK (role IN ('user','model','system')),
  quote_part TEXT,
  context_only INTEGER NOT NULL DEFAULT 0 CHECK (context_only IN (0,1)),
  PRIMARY KEY (session_id,position),
  UNIQUE (session_id,chat_id,msg_id),
  FOREIGN KEY (session_id) REFERENCES ai_sessions(id) ON DELETE CASCADE,
  FOREIGN KEY (chat_id,msg_id) REFERENCES ai_messages(chat_id,msg_id) ON DELETE RESTRICT
) WITHOUT ROWID, STRICT;
CREATE INDEX idx_ai_session_messages_chat_msg_context ON ai_session_messages(chat_id,msg_id,context_only);
CREATE TABLE ai_runs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  session_id INTEGER NOT NULL,
  request_chat_id INTEGER NOT NULL,
  request_msg_id INTEGER NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('pending','generated','delivered','delivery_failed','model_failed','persistence_failed')),
  provider TEXT NOT NULL CHECK (length(provider)>0),
  model TEXT NOT NULL CHECK (length(model)>0),
  requested_at INTEGER NOT NULL,
  completed_at INTEGER,
  input_tokens INTEGER CHECK (input_tokens IS NULL OR input_tokens>=0),
  output_tokens INTEGER CHECK (output_tokens IS NULL OR output_tokens>=0),
  cached_input_tokens INTEGER CHECK (cached_input_tokens IS NULL OR cached_input_tokens>=0),
  input_message_count INTEGER CHECK (input_message_count IS NULL OR input_message_count>=0),
  input_first_msg_id INTEGER,
  input_last_msg_id INTEGER,
  response_text TEXT,
  raw_payload BLOB,
  thought_signature TEXT,
  assistant_payload BLOB,
  assistant_payload_format TEXT,
  cache_expire_at INTEGER,
  candidate_state_json TEXT CHECK (candidate_state_json IS NULL OR json_valid(candidate_state_json)),
  response_chat_id INTEGER,
  response_msg_id INTEGER,
  error_code TEXT,
  error_message TEXT,
  UNIQUE (session_id,request_chat_id,request_msg_id),
  UNIQUE (response_chat_id,response_msg_id),
  CHECK ((response_chat_id IS NULL)=(response_msg_id IS NULL)),
  CHECK (input_first_msg_id IS NULL OR input_last_msg_id IS NULL OR input_first_msg_id<=input_last_msg_id),
  FOREIGN KEY (session_id) REFERENCES ai_sessions(id) ON DELETE CASCADE,
  FOREIGN KEY (request_chat_id,request_msg_id) REFERENCES ai_messages(chat_id,msg_id) ON DELETE RESTRICT,
  FOREIGN KEY (session_id,response_chat_id,response_msg_id)
    REFERENCES ai_session_messages(session_id,chat_id,msg_id) ON DELETE RESTRICT
) STRICT;
CREATE INDEX idx_ai_runs_session_requested ON ai_runs(session_id,requested_at DESC);
CREATE TABLE ai_session_provider_state (
  session_id INTEGER PRIMARY KEY,
  provider TEXT NOT NULL CHECK (length(provider)>0),
  state_version INTEGER NOT NULL CHECK (state_version>0),
  state_json TEXT NOT NULL CHECK (json_valid(state_json)),
  updated_at INTEGER NOT NULL,
  FOREIGN KEY (session_id) REFERENCES ai_sessions(id) ON DELETE CASCADE
) WITHOUT ROWID, STRICT;
CREATE TABLE ai_system_prompts (
  chat_id INTEGER NOT NULL,
  topic_id INTEGER NOT NULL CHECK (topic_id>=0),
  prompt TEXT NOT NULL,
  updated_at INTEGER NOT NULL,
  PRIMARY KEY (chat_id,topic_id)
) WITHOUT ROWID, STRICT;
-- CoC history is retained solely as an archive, never consumed by runtime handlers.
CREATE TABLE legacy_coc_attrs (
 user_id INTEGER NOT NULL, attr_name TEXT NOT NULL, attr_value TEXT NOT NULL,
 PRIMARY KEY(user_id,attr_name)
) WITHOUT ROWID, STRICT;
CREATE TABLE import_archives (
 source_table TEXT NOT NULL, row_key TEXT NOT NULL, payload TEXT NOT NULL CHECK(json_valid(payload)),
 PRIMARY KEY(source_table,row_key)
) WITHOUT ROWID, STRICT;
CREATE TABLE ai_user_settings (
 user_id INTEGER PRIMARY KEY, reactions_enabled INTEGER NOT NULL DEFAULT 1 CHECK(reactions_enabled IN (0,1)),
 updated_at INTEGER NOT NULL
) WITHOUT ROWID, STRICT;
CREATE TABLE traditional_facts (
 id INTEGER PRIMARY KEY, feature TEXT NOT NULL, chat_id INTEGER NOT NULL, topic_id INTEGER NOT NULL CHECK(topic_id>=0),
 request_msg_id INTEGER NOT NULL, reply_msg_id INTEGER, status TEXT NOT NULL CHECK(status IN ('success','failure')),
 context_text TEXT NOT NULL, created_at INTEGER NOT NULL,
 delivery_status TEXT NOT NULL CHECK(delivery_status IN ('delivered','failed'))
) STRICT;
CREATE INDEX idx_traditional_facts_reply ON traditional_facts(chat_id,topic_id,reply_msg_id);
CREATE TABLE ai_run_context (
 run_id INTEGER PRIMARY KEY REFERENCES ai_runs(id) ON DELETE CASCADE,
 snapshot TEXT NOT NULL CHECK(json_valid(snapshot))
) STRICT;
