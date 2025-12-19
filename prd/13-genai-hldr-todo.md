# Todo: genai handler extraction & execjs

- [ ] Finalize prompt template design and embed via `go:embed` with required placeholders
- [ ] Extract Gemini logic into `genai_hldr` (session cache, DB load/persist, prompt/tool assembly)
- [ ] Implement `//execjs` parsing and goja execution with time/memory/output limits; expose `reply()` for context-only outputs
- [ ] Return only “exec success/failure + brief summary” markers (no script/source/verbose output) into session/context
- [ ] Wire Telegram handler to new `genai_hldr` interfaces while preserving reactions/markdown/storage behavior
- [ ] Add tests for session hit/miss, prompt templating, execjs parsing/execution/error paths
- [ ] Run `gofmt`, `go mod tidy` (if deps change), and `go test ./...`
