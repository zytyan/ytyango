-- Only file metadata, never captions or ordinary chat text.
CREATE TABLE album_index (
  chat_id INTEGER NOT NULL,
  topic_id INTEGER NOT NULL,
  group_id TEXT NOT NULL,
  expires_at INTEGER NOT NULL,
  photo_only INTEGER NOT NULL CHECK(photo_only IN (0,1)),
  PRIMARY KEY(chat_id,topic_id,group_id)
) WITHOUT ROWID, STRICT;
CREATE INDEX idx_album_index_expiry ON album_index(expires_at);
CREATE TABLE album_files (
  chat_id INTEGER NOT NULL,
  topic_id INTEGER NOT NULL,
  group_id TEXT NOT NULL,
  msg_id INTEGER NOT NULL,
  file_id TEXT NOT NULL,
  file_unique_id TEXT NOT NULL,
  byte_size INTEGER,
  PRIMARY KEY(chat_id,topic_id,group_id,msg_id),
  FOREIGN KEY(chat_id,topic_id,group_id) REFERENCES album_index(chat_id,topic_id,group_id) ON DELETE CASCADE
) WITHOUT ROWID, STRICT;
CREATE INDEX idx_album_files_message ON album_files(chat_id,topic_id,msg_id);
