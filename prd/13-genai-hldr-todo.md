# Todo: genai handler extraction & execjs

- [x] Finalize prompt template design and embed via `go:embed` with required placeholders
- [x] Extract Gemini logic into `genai_hldr` (session cache, DB load/persist, prompt/tool assembly)
- [x] Implement `//execjs` parsing and goja execution with time/memory/output limits; expose `reply()` for context-only outputs
- [x] Return only “exec success/failure + brief summary” markers (no script/source/verbose output) into session/context
- [x] Wire Telegram handler to new `genai_hldr` interfaces while preserving reactions/markdown/storage behavior
- [ ] Add tests for session hit/miss, prompt templating, execjs parsing/execution/error paths
  - [x] Prompt templating and execjs parsing/execution/error notes
  - [ ] Session hit/miss coverage (needs lightweight DB harness)
- [x] Run `gofmt`, `go mod tidy` (if deps change), and `go test ./...`
- [ ] Add message search tool (keywords/user) exposed to genai_hldr and integrate into prompt/tools，结果需包含 user_id 与用户名
- [ ] Tests for message search tool and result shaping/limits（含 user_id/username 字段）
