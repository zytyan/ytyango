# HTTP Backend Ogen Migration TODO

- [x] 目录与 OpenAPI schema
  - [x] 在 `http/openapi.yaml` 定义搜索、批量用户信息（≤50）、头像接口；头像路由使用 `/users/{userId}/avatar?tgauth=...`，搜索仅 JSON 请求体。
  - [x] 为响应/请求体引入真实结构（SearchQuery/SearchResult、User.Name 输出、头像二进制），避免假数据占位。
  - [x] 保持目录仅为 `http`/`http/backend`，生成代码放置在受控子目录（如 `http/backend/ogen`）；提供 `make gen-http` 或 `go generate` 入口。
  - [x] 在 schema 里标注 Telegram tgauth 验证字段来源，配合 `sha256.Sum256(botToken)` 派生密钥。

- [ ] 后端入口与路由迁移
  - [x] 在 `http/backend` 使用 ogen router + `net/http` 初始化服务器，装配全局配置（Meili、bot token 验证 key）。
  - [ ] 保留必要的启动配置/日志，添加基础健康检查/版本信息（可选）；明确 ogen 生成代码与手写 handler 的分层。
  - [x] 入口需要暴露/注入 bot 供头像下载，并与主体服务的启动流程对齐（替换旧 `bothttp` 路由）。

- [ ] 搜索接口实现
  - [x] 复用 PRD 9.2.2 的 `meiliSearch` 逻辑，改造成 ogen handler（JSON 体→SearchQuery）并确保响应体关闭。
  - [x] 确保 `JsonInt64`/`GetLimit` 辅助函数可用，错误场景（缺失 chat、Meili 异常、非 JSON）返回 4xx/5xx。
  - [x] clamp `limit` 到 1..50，与 PRD 一致。
  - [ ] 添加测试覆盖成功路径、非 JSON/空内容、找不到 WebID 场景；若依赖 Meili/DB，添加最小可运行的 fixture/stub 而非假数据返回。

- [ ] 用户信息批量接口实现
  - [x] 从请求体解析用户 ID 列表，校验 1..50；空/超限返回 400。
  - [x] 调用 `g.Q` 查询真实用户数据并使用 `User.Name()` 返回；明确缺失用户的处理策略（全部 404 或局部过滤）。
  - [ ] 添加测试覆盖正常、0/超过 50、重复 ID、部分缺失的场景，使用真实结构/fixture 而非硬编码假用户。

- [ ] 用户头像接口实现
  - [x] 路由 `/users/{userId}/avatar?tgauth=...`，复用 PRD 9.1 的 `checkTelegramAuth` 验证逻辑；缺失/失败返回 401/403。
  - [x] 复用 `getUserProfilePhotoWebp` 与 `DownloadProfilePhoto`（支持 `sql.NullString`），确保缓存路径与目录存在，返回 404 当头像缺失。
  - [x] 校验 tgauth 与路径 `userId` 绑定，且验签密钥与 PRD（`sha256(botToken)`）一致。

- [ ] 清理与收尾
  - [x] 移除/停用未启用功能的处理与路由，确保仅核心接口暴露；标记遗留 `bothttp` 入口与迁移状态。
  - [x] 根据依赖变更运行 `go mod tidy`，格式化 `gofmt -w`，执行 `go test ./...`（记录外部服务依赖如 Meili/Telegram）。
  - [ ] 更新文档/README（如有）以指向新入口和生成流程；阶段性刷新 TODO 并 commit。
