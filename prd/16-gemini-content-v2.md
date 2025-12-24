# **Product Requirements Document**

## **产品名称：Gemini content v2 重构（gemini_ai）**

## **版本：v1.0**

## **撰写日期：2025-12-25**

---

## **1. 背景（Background）**

* 数据库已新增 `gemini_content_v2` / `gemini_content_v2_parts` 及扩展后的 `gemini_sessions`，但业务仍停留在旧的 `gemini_contents` 结构，历史生成逻辑自拼上下文、缺少对工具调用、缓存等能力的存储。
* 现有 Gemini 调用手工拼接 label，未使用官方 Chats / Caches 的上下文管理，无法保存工具调用结果或分片内容，导致历史无法复用、调试困难。
* 需求允许破坏性更新旧逻辑，目标是以 content v2 为唯一来源，充分利用 Gemini 自带的对话与缓存能力。

---

## **2. 目标（Objectives）**

* 以 `gemini_sessions` + `gemini_content_v2` / `gemini_content_v2_parts` 作为唯一读写路径，废弃旧表的业务依赖。
* 使用 Gemini 官方 Chats / Caches 管理对话上下文与长上下文缓存，避免手工拼接历史。
* 记录并复用 Gemini 的工具调用、代码执行、文件/多模态输入输出，数据库持久化每个 part 的结构化内容。
* 为维护者提供可观测的会话状态（缓存名称、TTL、过期标记、工具列表），便于调试与重放。

---

## **3. 非目标（Out of Scope）**

* 不做旧 `gemini_contents` 数据向 v2 的迁移清洗；历史对话可放弃或仅供只读。
* 不兼容旧的 “frozen” 会话或 `gemini_session_migrations` / `gemini_memories` 逻辑。
* 不新增跨平台入口（仅保持 Telegram 机器人），不实现多模型路由或多租户。

---

## **4. 用户角色（User Personas）**

### **Telegram 用户**

* 希望与机器人自然对话，包含文本、多模态、工具回复，且上下文连续。

### **维护者 / 开发者**

* 需要查看与排查单个对话的缓存与消息落库情况。
* 希望快速重置或重新开始会话，确认工具调用链路是否被记录。

---

## **5. 用户故事（User Stories）**

1. **作为用户，我发送文本或图片消息，机器人基于 Gemini Chats 直接读取上下文并回复，多模态内容和回复被完整记录到 v2 表中。**
2. **作为维护者，我可以查看某个 `session_id` 的缓存名称、TTL、过期状态，确认是否命中 Gemini Cache。**
3. **作为维护者，我能在数据库中看到模型返回的工具调用 / 工具响应 / 代码执行输出，支持复现或追踪问题。**
4. **作为用户，当上下文超出缓存 TTL 时，机器人自动刷新缓存并在新对话中继续，不需要我手动清理。**

---

## **6. 功能需求（Functional Requirements）**

| ID   | 描述 | 优先级 |
| ---- | ---- | ---- |
| FR-1 | 会话创建/恢复时，读取或写入 `gemini_sessions` 新字段（tools、cache_name、cache_ttl、cache_expired），使用 Gemini Chats 启动/续接对话。 | 高 |
| FR-2 | 入站消息解析（文本、媒体、引用、回复关系）写入 `gemini_content_v2`，并按 part 拆分存入 `gemini_content_v2_parts`，`seq` 逐条递增且唯一。 | 高 |
| FR-3 | 生成调用使用 Gemini Caches；命中缓存时复用 `cache_name`，过期或未命中时创建新缓存并更新 TTL / 过期标记。 | 高 |
| FR-4 | 模型返回的内容（含多段文本、inline data、file uri、function call/response、code execution、思考/thought）逐 part 落库，保持与原生 `genai.Content` 对齐。 | 高 |
| FR-5 | 生成配置中的工具列表写入 `gemini_sessions.tools`（JSON），与实际传给 Gemini 的工具定义一致。 | 中 |
| FR-6 | 提供会话重置/淘汰路径：当缓存失效或对话异常时，清空本地会话缓存并在 DB 记录新的 cache 状态。 | 中 |
| FR-7 | 日志与可观测性：对每次生成记录请求 ID、模型名、缓存命中信息、耗时；错误时写入 DB/日志方便追踪。 | 中 |
| FR-8 | 单次回复前将 Telegram 消息临时缓存到内存，回复成功后以事务落库 v2 表；失败时回滚避免空洞 `seq`。 | 中 |
| FR-9 | 兼容多模态：支持图片、贴纸等 telegram 输入对应 inline data；模型输出文件 URI / inline data 也要落库。 | 低 |

---

## **7. 非功能需求（Non-functional Requirements）**

* 数据一致性：同一 session 内 `seq` 自增且唯一，落库使用事务，防止部分成功。
* 性能：优先命中 Gemini Cache 减少上下文拼接，历史加载限制条数可配置，避免超时。
* 可维护性：与 `genai` 官方类型一一映射，减少自定义结构；表字段保持直观对应。
* 可测试性：提供会话创建、缓存命中/失效、工具调用落库的单元/集成测试。

---

## **8. 技术方案（Tech Design Summary）**

* 使用 `genai` 的 Chats / Caches API：创建/续接 Chat 时绑定 `cache_name`（若存在且未过期），并将工具配置直接传递给模型，无需手工拼接历史文本。
* 数据映射：将 `genai.Content` 与 `Part` 映射到 `gemini_content_v2` / `gemini_content_v2_parts`，未知字段放入 `x_user_extra` JSON，保持未来兼容。
* 会话管理：`gemini_sessions` 存储工具列表、缓存 TTL、过期时间；缓存失效后写 `cache_expired` 并生成新缓存名。
* 入出站管道：Telegram 消息 -> 结构化 part（文本/inline data/file URI），回复内容同样分片落库；事务包裹写入，失败回滚。
* 配置与限制：保留生成模型名配置，可调整历史加载上限；cache TTL 默认值落在配置或常量中。
* 观察性：记录请求/响应摘要（日志或 DB 字段），输出 cache 命中与耗时，便于调试。

---

## **9. 数据结构（Data Models）**

* `gemini_sessions(id, chat_id, chat_name, chat_type, tools JSON_TEXT, cache_name TEXT, cache_ttl INTEGER, cache_expired INTEGER)` — 记录会话、工具、缓存状态。
* `gemini_content_v2(id, session_id, role, seq, x_user_extra JSON_TEXT)` — 会话消息主表，`seq` 为会话内顺序号。
* `gemini_content_v2_parts(id, content_id, part_index, text, thought, thought_signature, inline_data, inline_data_mime, file_uri, file_mime, function_call_name, function_call_args, function_response_name, function_response, executable_code, executable_code_language, code_execution_outcome, code_execution_output, video_start_offset, video_end_offset, video_fps, x_user_extra)` — 细粒度 part 存储。

---

## **10. 里程碑（Milestones）**

| 时间 | 目标 |
| ---- | ---- |
| Day 1 | 完成方案设计与数据映射定义，确认缓存/工具策略与落库字段。 |
| Day 2 | 改造 gemini_ai 逻辑接入 Chats/Caches，完成入站/出站落库与事务处理。 |
| Day 3 | 补充测试（缓存命中/失效、工具调用、多模态落库）与日志观测，验证通过后准备上线。 |

---

## **11. 使用说明（How to Use）**

* Telegram 用户直接对话，机器人自动基于 `session_id` 选择或创建 Gemini Chat，命中缓存则复用，过期则重建。
* 维护者可通过 DB 查询 `gemini_sessions` 查看当前缓存名、TTL、过期标记与工具列表，必要时清空行以强制新会话。
* 回复内容与工具调用均在 v2 表可见，按 `session_id` + `seq` 追踪整条对话。若缓存异常，重置会话后继续对话即可。 
