# **Product Requirements Document**

## **产品名称：Telegram 消息持久化存档**

## **版本：v1.0**

## **撰写日期：2025-12-13**

---

## **1. 背景（Background）**

当前 `SaveMessage` 仅向 MeiliSearch 推送部分文本，无法保证持久化与回溯：

* 群组可选“保存消息”开关，但并未真正落库，删除或回溯时无数据可查。
* 新增的 `saved_msgs` / `raw_update` / `edit_history` schema 已准备好 WITHOUT ROWID 与触发器，但还缺落地代码与行为约束。
* 需要保留完整元信息（线程、转发来源、媒体 file_id 等）以支撑后续检索、审计与调试。

---

## **2. 目标（Objectives）**

* 将开启保存的聊天消息可靠写入 `saved_msgs`，覆盖文本、实体、媒体及转发信息。
* 持久化 Telegram 原始 Update（可选裁剪），支持问题复现与审计。
* 处理消息编辑：更新内容并记录 `edit_history`，保持时间序列。
* 提供按 `chat_id + message_id` 读取接口，供后续 HTTP/机器人功能复用。

---

## **3. 非目标（Out of Scope）**

* 不实现搜索/浏览 UI 或 HTTP API（仅落库与读取接口）。
* 不在本迭代迁移旧的 FTS / Meili 索引或补录历史数据。
* 不新增群聊配置项（沿用现有 `SaveMessages` 开关即可）。

---

## **4. 用户角色（User Personas）**

### **Bot 运维/开发者**

* 需要完整消息与原始 Update 以便排障、审计。
* 希望写入失败不影响机器人主流程。

### **群管理员/群主**

* 开启保存后，希望消息（含媒体信息）可被后续查询或转存。

### **产品/数据使用方**

* 期望未来可基于存档数据构建搜索、统计或标签能力。

---

## **5. 用户故事（User Stories）**

1. 作为运维，我为某群打开“保存消息”后，新消息（含频道、话题消息）会落入 SQLite，记录发送者、时间、文本和媒体标识。
2. 作为开发者，我在收到转发或回复时，能在存档里看到原消息 ID、来源 chat/user 信息及 media_group_id，方便回溯上下文。
3. 作为开发者，我编辑一条已存档消息时，`saved_msgs` 会更新文本，且 `edit_history` 记录旧版本以供追踪。
4. 作为维护者，我遇到异常时可以查到对应 Update 的原始 JSON（或精简版），确认 Telegram 载荷。

---

## **6. 功能需求（Functional Requirements）**

### **6.1 消息写入 saved_msgs**

| ID | 描述 | 优先级 |
| --- | --- | --- |
| FR-1 | 仅当全局允许且群配置 `SaveMessages=true` 时写入；不阻塞主消息流程，失败仅记日志。 | 高 |
| FR-2 | 新增或首次看到的消息按 `(chat_id, message_id)` 写入一行，填充 `from_user_id`/`sender_chat_id`、`date`、`message_thread_id`、`reply_to_*`、`via_bot_id`、`media_group_id`、`edit_date`（如有）。 | 高 |
| FR-3 | 记录转发来源：`forward_origin_name`、`forward_origin_id`，区分用户/频道来源。 | 中 |
| FR-4 | 存储可见文本：优先 `message.Text`，无正文时使用 `Caption`；`entities_json` 保存 Telegram entities/caption_entities 的结构化 JSON。 | 高 |
| FR-5 | 存储媒体主键：选择最能代表消息的 `media_id`/`media_uid` 与 `media_type`（photo/video/document/voice/video_note/animation/sticker/story 等），并保存 `media_group_id`。 | 高 |
| FR-6 | 对非文本/媒体类负载（poll/quiz/dice/location/contact 等）将核心字段写入 `extra_data`（JSONB），并标记 `extra_type`。 | 中 |
| FR-7 | 遇到重复 `(chat_id, message_id)` 插入时不影响主流程：若记录已存在则跳过或覆盖（以 DB 约束为准），错误需降级为 warn。 | 中 |

### **6.2 原始 Update 与关联表**

| ID | 描述 | 优先级 |
| --- | --- | --- |
| FR-8 | 将 Telegram 原始 Update（或裁剪后的重要字段）序列化为 JSON，写入 `raw_update`，关联 `chat_id`/`message_id`（允许为空）。 | 中 |
| FR-9 | 确保 `raw_update` 写入失败不影响主流程，必要时限制/截断体积（如 >256KB 仅存摘要）。 | 低 |

### **6.3 编辑/更新同步**

| ID | 描述 | 优先级 |
| --- | --- | --- |
| FR-10 | 处理 `edited_message`/`edited_channel_post`：同步更新 `text`、`entities_json`、`edit_date`；触发器写入 `edit_history`，无需额外业务逻辑。 | 高 |
| FR-11 | 提供按 `chat_id + message_id` 查询接口（封装 `GetSavedMessageById`），供其他 handler 或 HTTP 层复用。 | 中 |

### **6.4 可观测性与稳健性**

| ID | 描述 | 优先级 |
| --- | --- | --- |
| FR-12 | 记录日志字段至少包含 `chat_id`、`message_id`、消息类型、耗时/错误；配置慢查询阈值沿用 q 包 logger 方案。 | 中 |
| FR-13 | 插入/更新逻辑在 goroutine 中运行，避免阻塞消息调度；尊重 ctx 取消但不中断主线程。 | 高 |

---

## **7. 非功能需求（Non-functional Requirements）**

* **性能**：单次写入期望 <20ms，使用预编译语句与共享 DB 连接；对批量（相册）不显著拖慢消息处理。
* **可靠性**：写入异常不影响其他 handler；数据库保持 WAL 与 STRICT/NO ROWID 约束。
* **安全**：日志避免输出完整文本/原始 JSON；敏感字段可截断或摘要化。
* **可维护性**：代码需单元测试覆盖主要类型映射；依赖变更需 `go mod tidy` 并 `gofmt`。

---

## **8. 技术方案（Tech Design Summary）**

* 在 `globalcfg` 初始化阶段准备 `msgs.Queries`（与现有 `q.PrepareWithLogger` 风格一致），共享同一 SQLite 连接与 zap logger。
* 新增转换层：从 `gotgbot` 的 Message/Update 构造 `CreateNewMessageParams`，负责选择文本/媒体/extra_type，序列化 entities 与 extra_data。
* 更新 `SaveMessage`/相关 handler：在 goroutine 内调用 `CreateNewMessage`，并按需写入 `raw_update`；捕获重复主键错误并降级日志。
* 处理编辑：在 `SaveMessage` 或独立 handler 中监听 edited updates，调用 `UpdateMessageText` 并更新相关字段，依赖触发器写入 `edit_history`。
* 预留读取接口：封装 `GetSavedMessageById` 便于后续 HTTP/web 使用。

---

## **9. 数据结构（Data Models）**

### **saved_msgs（核心）**

```sql
PRIMARY KEY (chat_id, message_id) WITHOUT ROWID;
-- 字段：from_user_id/sender_chat_id、date(INT_UNIX_SEC)、forward_origin_*、message_thread_id、reply_to_*、via_bot_id、edit_date、media_group_id、text、entities_json、media_id/media_uid/media_type、extra_data/extra_type
```

### **raw_update**

```sql
id INTEGER PRIMARY KEY,
chat_id INTEGER,
message_id INTEGER,
raw_update BLOB_JSONB,
INDEX(id, chat_id)
```

### **edit_history**

```sql
PRIMARY KEY(chat_id, message_id, edit_id) WITHOUT ROWID;
text TEXT -- 由 trigger_on_edit_message 自动插入旧文本
```

---

## **10. 里程碑（Milestones）**

| 时间 | 目标 |
| --- | --- |
| Day 1 | PRD 评审通过，明确字段映射与异常策略。 |
| Day 2 | 完成写入/更新实现，处理主要消息与媒体类型，自测基础案例。 |
| Day 3 | 增补原始 Update 写入、日志与测试，`go test ./...` 验证。 |

---

## **11. 风险（Risks）与对策（Mitigations）**

| 风险 | 影响 | 对策 |
| --- | --- | --- |
| Telegram 消息类型多样，未覆盖类型可能导致丢字段 | 数据不完整 | 统一 extra_data 兜底，并在日志中标记未覆盖类型，逐步补齐。 |
| 原始 Update 体积过大导致写入失败或膨胀 | 数据库膨胀/慢 | 对大于阈值的 JSON 做截断/哈希摘要，仅保存必要字段。 |
| 主键冲突或并发写入导致错误 | 影响落库成功率 | 捕获 UNIQUE 约束错误并降级处理；必要时使用 UPSERT 或 skip 策略。 |

---

## **12. 验收标准（Acceptance Criteria）**

* 收到文本/媒体/转发/回复消息时，`saved_msgs` 记录含正确 ID、时间、文本/实体、媒体字段与关联信息。
* 收到编辑事件后，`text` 与 `edit_date` 被更新，`edit_history` 追加一条旧文本记录。
* 原始 Update 被写入 `raw_update` 或被安全地截断；写入失败不影响机器人运行。
* `go test ./...` 通过（如因外部依赖失败需说明原因），`gofmt`、`go mod tidy` 达到一致性。
