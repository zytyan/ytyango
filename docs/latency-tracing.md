# AI 卡顿分析与跟踪（2026-10-13）

检查 TS 前置 e86895f、封存 Go archive/go/20261013/codex/ai-db-v2 d395a28。 只读比较 TS
生产切换之后的 AI Run，不把 Go 历史请求当作 TS 延迟。

已有证据：消息发出到 Run 开始最大约 21 秒，Run 开始到生成落库最大 88 秒。 会话 10441 的 88
秒请求输出 3968 Token，其中推理 3268；实际返回模型为 gemini-3.8-flash， 配置模型名仍为
gemini-3.7-flash。其他 35–59 秒请求也有较多推理。
这些记录只能证明存在生成前等待和长生成，无法区分网络、排队、审核、锁、重试和交付。

## 跟踪范围

只跟踪明确 AI 触发、命令、按钮、inline 查询及实际启用的自动工具/审核候选。
普通聊天不生成跟踪记录，不保存正文、caption、用户姓名、聊天名称、媒体、链接、凭据或原始错误。 收到
polling 批次时仅临时记录 update ID 和时间，派发后删除，最多 2048 项。
跟踪不写数据库，不调用模型，也不改变取消、重试、会话事务或 Telegram 副作用。

单请求包含 update_id、类型、开始 UTC 时间，AI 请求增加 session_id/run_id/provider/model。
时间字段均为毫秒：

- transport_lag_ms：Telegram 消息时间到本地 getUpdates 返回；消息时间只有秒精度，
  还受本机时钟影响，因此不能全归因于 Telegram 网络。按钮不拿旧菜单日期当作点击时间。
- runner_queue_ms：本地收到更新后到 handler 启动，定位 16 个 runner 并发槽被占用的情况。
- elapsed_ms：handler 内总耗时；since_received_ms 同时包含 runner 等待。
- stages：传统处理、相册聚合、AI 会话锁、表情回应、媒体准备、本地历史、显式缓存、
  供应商响应头/响应体、按协议的尝试次数、重试退避、AI 交付、Telegram 方法、
  下载队列/singleflight、外部进程与图片审核队列的次数/耗时/最大耗时/失败次数。
- waiting：当前尚未完成的阶段及已等待时间，可在请求还没结束时诊断。

stages 存在父子嵌套及并行（如 typing），不能将所有阶段耗时直接相加。 开始前约 20 秒可能来自 Telegram
接收、传统审核、会话锁或反应请求；需跟踪辨别。 同一 AI 会话串行是历史与 provider
状态一致性要求；等待本身不能视为死锁。 15 分钟生成期限、五次总尝试和 SDK maxRetries=0
保持用户此前确认的策略。 普通消息头显示 Telegram 原始发送时间，收到晚或排队时不等于生成时刻。

## 查看方式

受既有内部 HTTP 鉴权保护的 GET /debug/latency 返回当前请求、最近 100 条完成记录、
最近事件循环最大延迟及周期 stats.flush 的耗时统计。记录仅在进程内，重启清空。 journal 的
`[runtime] latency` 为单行 JSON；完成超过 5 秒或接收延迟超过 10 秒用 warn， 其余用 info，遵循
runtime 日志等级。每 30 秒对正在运行超过 10 秒的请求或 事件循环延迟超过 500ms 输出
`[runtime] latency_active`，避免挂起请求无记录。

测试覆盖并发跟踪隔离、等待中的会话锁可见、错误原样传播且不记录错误正文、
有限历史、普通聊天不跟踪及诊断接口鉴权。独立编译 smoke 另验证实际 polling 到
AI/重试/交付各阶段及受保护的诊断读取。

## 线上样本和即时反馈修复

跟踪版本 5b7be59 已上线并只读观察两分钟，取得一轮会话 10461： 接收差约 783ms、runner 等待
2.7ms、表情请求 102ms、typing 最慢 36ms、 供应商响应头等待 5415ms、交付 69ms，总处理约 6059ms。
该轮主要慢在供应商；没有会话锁等待、重试或事件循环卡死。 观察窗口事件循环最大延迟
40.9ms，周期统计写入最大 19.6ms；这不是严重卡顿的复现。 一次旧按钮收到 400 query too
old，后续按钮正常；API 返回 ok=false 的情况也标为阶段失败， 保留 grammY
原有错误转换和处理，跟踪不吞掉错误。

用户进一步指出 reaction 与 typing 也晚。代码确认：旧 typing 位于传统处理和相册等待之后， 旧 👀
reaction 位于会话锁之后，且 awaited，因此最长可被 Telegram 默认 130 秒超时拖住。 修复在确认普通 AI
输入触发、非已交付重复请求后立即发起 typing 和用户启用的 👀 reaction，
覆盖传统工作、相册等待和会话锁阶段。reaction 异步 best effort、最多等待 2 秒且不重试，
完成/失败后中止未完成状态请求；失败表情也有 2 秒上限。typing 每次 API 请求限时 2 秒，
慢或挂起的状态请求不会阻止后续续期。没有改变生成、事务或交付重试策略。
临时输入、非触发消息、已交付重复请求仍不发状态，不采集普通聊天。 诊断增加
pending_dispatch_count/oldest_pending_dispatch_ms，帮助识别 runner 派发阻塞。
回归模拟传统链接转换挂起和 reaction 挂起，确认 typing/reaction 已先发起、 模型仍可完成、reaction
被取消、仅调用一次；另覆盖 typing 超时后的续期且不重叠。
