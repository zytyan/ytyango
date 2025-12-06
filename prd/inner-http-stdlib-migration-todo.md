# Inner HTTP 服务标准库迁移 TODO

- [ ] 实现 `net/http` 版本的 `inner_http.go`，移除 Gin 依赖（仅该文件），保持现有路由行为兼容。
- [ ] 为 `/mars-counter` 和 `/dio-ban` 使用 `json.Decoder` + `http.MaxBytesReader` 解析请求体，沿用现有 400/200 行为。
- [ ] 提供 `/loggers`（GET）和 `/loggers/:name/:level`（PUT）路由，保持纯文本输出与未找到 logger 的提示。
- [ ] 根路径返回路由列表文案，与现有输出一致。
- [ ] 接入 zap 访问日志中间件，记录方法、路径、状态码、耗时；加入 panic 恢复并返回 500。
- [ ] 基于 `net/http/pprof` 暴露 pprof 端点（`/debug/pprof/*`），与其他路由共存。
- [ ] 支持 `BOT_INNER_HTTP` 配置：空值默认 `127.0.0.1:4019`；`OFF` 时跳过监听并记录日志；不可解析值直接 panic。
- [ ] 更新 `formatLoggers`/logger 处理逻辑以适配标准库 handler（路径参数解析等）。
- [ ] 添加测试覆盖：各路由的 200/400 行为、未知 logger 提示、环境变量开关（OFF/自定义/非法）、pprof 基本可达性。
- [ ] 运行 `gofmt -w`、如依赖变更执行 `go mod tidy`，并执行 `go test ./...`（记录外部依赖导致的失败原因如有）。
