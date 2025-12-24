# TODO - Inner HTTP 数据库备份下载

- [x] Implement `/backupdb` GET handler using sqlite3 Backup API to snapshot main/msg databases with `db=main|msg|all` and optional token validation.
- [x] Generate manifest metadata (timestamp, source paths, sizes) and stream zip response with proper headers; ensure temp files cleaned up and errors logged.
- [x] Add unit tests in `handlers/inner_http_test.go` covering route success (all + scoped) and auth failures.
- [ ] Verify formatting (`gofmt`) and tests (`go test ./...`); document any known test skips/failures.
  - [x] gofmt -w handlers/inner_http.go handlers/inner_http_test.go
  - [ ] go test ./... (blocked by handlers/gemini_ai.go: content.ThoughtSignature.String type mismatch)
