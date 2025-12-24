package migrate

// MigrationsMain holds migrations for the main database.
var MigrationsMain = []Migration{
	{
		Version: 1,
		Name:    "add gemini content v2 tables",
		Up: []Step{
			{
				Description: "create gemini_content_v2 and gemini_content_v2_parts",
				SQL: []string{
					`CREATE TABLE IF NOT EXISTS gemini_content_v2 (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id INTEGER NOT NULL REFERENCES gemini_sessions (id) ON DELETE CASCADE,
    role TEXT NOT NULL,
    seq INTEGER NOT NULL,
    x_user_extra JSON_TEXT,
    UNIQUE(session_id, seq)
);`,
					`CREATE INDEX IF NOT EXISTS idx_gemini_content_v2_session ON gemini_content_v2(session_id);`,
					`CREATE TABLE IF NOT EXISTS gemini_content_v2_parts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    content_id INTEGER NOT NULL REFERENCES gemini_content_v2 (id) ON DELETE CASCADE,
    part_index INTEGER NOT NULL,
    text TEXT,
    thought INT_BOOL NOT NULL DEFAULT 0 CHECK (thought IN (0,1)),
    thought_signature BLOB,
    inline_data BLOB,
    inline_data_mime TEXT,
    file_uri TEXT,
    file_mime TEXT,
    function_call_name TEXT,
    function_call_args JSON_TEXT,
    function_response_name TEXT,
    function_response JSON_TEXT,
    executable_code TEXT,
    executable_code_language TEXT,
    code_execution_outcome TEXT,
    code_execution_output TEXT,
    video_start_offset TEXT,
    video_end_offset TEXT,
    video_fps REAL,
    x_user_extra JSON_TEXT,
    UNIQUE(content_id, part_index)
);`,
					`CREATE INDEX IF NOT EXISTS idx_gemini_content_v2_parts_content ON gemini_content_v2_parts(content_id);`,
				},
			},
		},
		Down: []Step{
			{SQL: []string{
				`DROP TABLE IF EXISTS gemini_content_v2_parts;`,
				`DROP TABLE IF EXISTS gemini_content_v2;`,
			}},
		},
	},
}

// ExpectedSchemaVersionMain tracks the latest expected schema version for main DB.
const ExpectedSchemaVersionMain = 1
