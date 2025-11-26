# Repo Agent Instructions （更新于目录调整后）

## 基础要求
- Go 版本：`go 1.25`（模块名 `main`）。
- 依赖变更后运行 `go mod tidy`；所有 Go 源码提交前执行 `gofmt -w`。
- 提交前跑 `go test ./...`；若测试依赖外部服务导致失败，请在提交说明中标注。
- 常用构建：`go build -tags=jsoniter -ldflags "-X 'main.compileTime=$(date "+%Y-%m-%d %H:%M:%S")'" -o build/ytyan-go`.

## 目录速览（与生成流程相关）
- `main.go`：Telegram 入口，启动 HTTP 后端、注册全部命令/回调处理器。
- `globalcfg/`：配置加载、日志、全局 SQLite 连接；`globalcfg/q` 为 SQLC 生成代码；`h` 内封装 Chat/Video 等助手。
- `helpers/`：业务工具库（Azure OCR/内容审核、bilibili BV/AV 转换、CoC 掷骰、汇率、图像处理/WebP、ytdlp、数学解析）；部分目录含单元测试。
- `myhandlers/`：所有 Telegram 指令与消息处理实现，依赖 `helpers` 与 `globalcfg`。
- `http/`  
  - `backend/`：OpenAPI 后端，`botapi.yaml` 为接口定义，`botapi/` 下为 ogen 生成的 server 代码。  
  - `frontend/`：Vue3 + Vite SSG 前端，`src/` 业务代码，`src/api/schema.ts` 由 OpenAPI 生成；`dist/` 为构建产物。
  - `generate.go`：`go:generate` 钩子，刷新后端 ogen 代码与前端 TS Schema。
- `sql/`：SQLite schema & 查询；`sqlc.yaml` 指向 wasm 插件 `sql/plugins/sqlc-gen-go.wasm`，输出到 `globalcfg/q`。
- `build/`：运行时产物与示例数据库 `ytyan_new.db`；`manage.sh` 默认将二进制放这里。
- `scripts/migrate_sqlite.py`：数据库迁移小工具。

## 生成/更新步骤
- 更新 OpenAPI（接口或前端类型变更）：在仓库根目录运行 `go generate ./http`（需要 `ogen` 可执行与 `npx openapi-typescript`，Node 依赖见 `http/frontend/package.json`）。
- 更新数据库访问层（SQL 变更）：运行 `sqlc generate`（需 sqlc v2，自动加载仓库内 wasm 插件）。
- 前端构建：`npm install --prefix http/frontend`（若依赖变），`npm run build --prefix http/frontend` 产出到 `http/frontend/dist`。
- 后端/整体构建：`go build ...` 或使用 `./manage.sh build`（支持自动重启 systemd 服务）。

## 运行配置（环境变量）
- `GOYTYAN_CONFIG`：必需，指向 YAML 配置；字段含 `bot-token`、`god`、`my-chats`、`meili-config`、`test-mode`、`content-moderator`、`ocr`、`tg-api-url`、`drop-pending-updates`、`tmp-path`、`database-path`、`gemini-key` 等。
- `GOYTYAN_LOG_FILE`：日志文件；若未设置则输出到 stderr。`GOYTYAN_NO_STDOUT=1` 可关闭标准输出。
- 其他：`TZ`（时区），`GOYTYAN_CONFIG` 内的路径应与 `build/` 或实际部署路径一致。

## 开发注意事项
- HTTP 后端安全校验在 `test-mode=true` 时放宽，便于本地/CI；正式环境务必关闭。
- 新增 Telegram 命令/回调需在 `myhandlers` 中实现并在 `main.go` 注册；若涉及 API，请同步更新 `botapi.yaml` 并重新生成。
- 修改 SQL、OpenAPI 或生成代码后，请确保重新运行相关生成指令再提交。
