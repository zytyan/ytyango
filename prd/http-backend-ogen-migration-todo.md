# HTTP Backend Ogen Migration TODO

- [ ] 目录与 OpenAPI schema
  - [ ] 在 `http/openapi.yaml` 定义搜索、批量用户信息（≤50）、头像接口；头像缺失返回 404，tgauth 作为查询参数；明确仅 JSON 搜索请求。
  - [ ] 保持目录仅为 `http`/`http/backend`，不创建 `frontend` 目录；生成代码放置在受控子目录（如 `http/backend/ogen`）。
  - [ ] 添加生成命令（Makefile/脚本或 `go generate`）调用已安装的 ogen；更新 `.gitignore` 以忽略不应提交的生成物（如需要）。

- [ ] 后端入口与路由迁移
  - [ ] 将服务器入口迁移到 `http/backend`，使用 ogen 生成的 router + `net/http`，移除 `bothttp`/Gin 依赖。
  - [ ] 保留必要的启动配置/日志，添加基础健康检查/版本信息（可选）。

- [ ] 搜索接口实现
  - [ ] 实现仅接收 `application/json` 请求体的搜索处理；拒绝 multipart/GET/空 Content-Type，返回 400。
  - [ ] 提供假数据或内存存根返回结构化搜索结果，包含分页字段（如需要）。
  - [ ] 添加测试覆盖成功路径与错误路径（非 JSON、无效请求体）。

- [ ] 用户信息批量接口实现
  - [ ] 支持批量用户 ID 列表解析，校验数量上限 50（含边界）；超出或空列表返回 400。
  - [ ] 返回用户基本信息字段（id、name、username），使用假数据或存根。
  - [ ] 添加测试覆盖正常、0/超过 50、重复 ID 等场景。

- [ ] 用户头像接口实现
  - [ ] 路由 `/users/{userId}/avatar?tgauth=...`，校验 tgauth；缺失返回 401/403。
  - [ ] 返回头像二进制；不存在返回 404；可提供假头像数据/占位生成。
  - [ ] 添加测试覆盖认证失败、头像缺失、成功返回二进制响应。

- [ ] 清理与收尾
  - [ ] 移除/停用未启用功能的处理与路由，确保仅核心接口暴露。
  - [ ] 根据依赖变更运行 `go mod tidy`，格式化 `gofmt -w`，执行 `go test ./...`（记录外部依赖导致的失败如有）。
  - [ ] 更新文档/README（如有）以指向新入口和生成流程。
