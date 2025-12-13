# Message Storage DB TODO

- [ ] 全局初始化
  - [ ] 在 `globalcfg` 中准备 `msgs.Queries`（共享 logger/SlowQueryThreshold），复用同一个 SQLite 连接。
  - [ ] 确认 schema/sqlc 生成产物在初始化时加载（含 `INT_UNIX_SEC` 映射）。

- [ ] SQL 与查询扩展
  - [ ] 为 `raw_update`/`edit_history` 写入补充 sqlc 查询（插入/查询），必要时添加 UPSERT 以处理重复。
  - [ ] 生成并验证新的 sqlc 输出，保持自定义 plugin 选项一致。

- [ ] 消息落库逻辑
  - [ ] 编写从 `Update/Message` 到 `CreateNewMessageParams` 的映射：文本/Caption、entities_json、线程/回复/转发、media_group_id。
  - [ ] 选择媒体主键（photo/video/document/voice/video_note/animation/sticker/story 等），填充 `media_id`/`media_uid`/`media_type`；其他类型落入 `extra_data`/`extra_type`。
  - [ ] 在 `SaveMessage` 路径中调用 DB 写入，尊重 `SaveMessages` 开关，处理重复主键降级为 warn。

- [ ] 编辑与更新
  - [ ] 处理 `edited_message`/`edited_channel_post`：更新文本/实体/`edit_date`；验证触发器写入 `edit_history`。
  - [ ] 暴露 `GetSavedMessageById` 封装方法，供后续 handler/HTTP 使用。

- [ ] 原始 Update 存储
  - [ ] 序列化（或裁剪）原始 Update，写入 `raw_update`，并对超大载荷做截断/摘要。
  - [ ] 确保写入失败不影响主流程，记录必要日志字段。

- [ ] 测试与验证
  - [ ] 添加单元/集成测试覆盖文本、媒体、编辑路径，验证存储字段。
  - [ ] 运行 `go test ./...`，在结果中注明外部依赖或跳过理由。
