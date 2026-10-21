-- NULL means permanent suspension; deleting the row resumes the session.
CREATE TABLE ai_session_suspensions (
  session_id INTEGER PRIMARY KEY REFERENCES ai_sessions(id),
  until_at INTEGER CHECK(until_at IS NULL OR until_at > 0)
);
