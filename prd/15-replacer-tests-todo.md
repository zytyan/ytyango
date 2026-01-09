# Todo: replacer tests

- [x] Review helpers/replacer implementation against handlers/gemini_ai.go usage
- [x] Define test cases for template parsing, escaping, and invalid variables
- [x] Add tests covering time/chat/bot variables with fixed ReplaceCtx
- [x] Fix replacer issues found during tests and keep todo updated
- [x] Run gofmt and go test ./... (note any external dependency failures)
  - [ ] go test ./... fails: panic unknown datatype for gemini_sessions.frozen: \"INT_BOOL\" (globalcfg_init_for_tests.go:56)
