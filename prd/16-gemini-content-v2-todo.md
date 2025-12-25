# Gemini content v2 TODO

- [ ] 需求确认
  - [ ] 评审并锁定 PRD（目标、范围、数据映射与缓存策略）。

- [x] 设计与数据映射
  - [x] 确认 `gemini_sessions` / `gemini_content_v2` / `gemini_content_v2_parts` 字段与 genai `Content/Part` 的映射方案（含 x_user_extra 扩展）。
  - [x] 定义 seq/part_index 生成策略与事务边界，避免空洞与乱序。

- [x] 会话与缓存
  - [x] 接入 Gemini Chats/Caches：创建/续接 Chat，缓存命中/失效时更新 `cache_name`、`cache_ttl`、`cache_expired`。
  - [x] 将工具配置写入 `gemini_sessions.tools`，保证与传给模型的工具列表一致。
  - [x] 提供会话重置/淘汰路径，清理本地缓存并落库新的 cache 状态。

- [x] 入站消息落库
  - [x] 解析 Telegram 文本/媒体/引用为 v2 消息与 part，按 seq 递增写入，使用事务确保失败回滚。

- [x] 生成与出站落库
  - [x] 使用 Chats/Caches 发起生成，记录请求 ID / 模型名 / 缓存命中信息。
  - [x] 将模型返回内容（文本、多模态、工具调用/响应、代码执行、thought）拆分 part 落库。

- [ ] 观测与错误处理
  - [ ] 增加日志/指标：耗时、缓存命中、错误原因；错误时保证事务回滚并打标可重试状态。

- [x] 配置与限流
  - [x] 提供模型名、历史加载上限、默认 cache TTL 等配置项或常量。

- [ ] 测试
  - [ ] 单元/集成测试覆盖：缓存命中/过期路径、工具调用持久化、多模态输入输出落库、事务回滚。

- [x] 收尾
  - [x] gofmt / go test ./... 记录结果，更新文档或帮助信息（如有）。 
