# 简化架构与 AI SDK 评估

## 基线

本次读取与远程核对的最新代码：

- TS `codex/retain-features`：`b991db7396befea259ae139dd422546cfa557d81`。
- Go `codex/ai-db-v2`：`9b52b20188c592f736987eca97d9ece65aa43a41`（当前最新开发提交）。
- Go `master`：`7a73f0e79e7698f94439382e839618f085b81ef3`；开发分支另有极小数格式修复。

不再以旧的 `b6376...` 冻结提交制定目标。上面是本次核对记录，不是新的永久冻结基线；
每次实现切片应刷新相关分支 HEAD。原 Go 工作树未 checkout 或合并。

## 结论

统一多家模型的候选是 **Vercel AI SDK**，不是旧的 Google Vertex AI SDK。
两者名称不同，用户所指名称尚待明确；本报告同时覆盖，不据此自动切换云平台或运行时依赖。

旧 `@google-cloud/vertexai` 生成式 AI 接口已进入官方退役迁移范围，官方方向是 `@google/genai`。
Google Gen AI SDK 可访问 Gemini Developer API/Vertex AI，但不是现有独立 DeepSeek、Sub2API
接口的跨厂商通用协议。 不为了“统一 SDK”把现有 API 供应商和计费渠道搬到 Vertex AI。
依据：[Google 迁移说明](https://docs.cloud.google.com/vertex-ai/generative-ai/docs/deprecations/genai-vertexai-sdk)、
[Google Gen AI SDK](https://github.com/googleapis/js-genai)。

Vercel AI SDK 的统一生成调用有 Google/DeepSeek provider，适合消除本项目的手写协议与响应转换；
业务状态仍由本项目保存。无需引入 Agent、UI hooks、Next.js、服务容器或多层 provider 注册框架。
依据：[providers](https://ai-sdk.dev/docs/foundations/providers-and-models)。

## 当前具体的抽象问题

- `src/shared/types.ts` 把 Telegram、media、process、clock 的行为端口集中在一起，92 行；
  单实现端口可改为具体对象或少量可替换函数，数据结构类型不是删除目标。
- `GrammyTelegramGateway` 116 行，许多方法只是转发 grammY；普通命令直接用 ctx/API。 保留 RichMessage
  窄扩展、必要消息身份/显式事实关联，不复制整套 Telegram API。
- `AiProvider` 和 `DeepSeekProvider/GeminiProvider` 是自定义行为契约；`providers.ts` 478 行，
  手写编码/HTTP/响应解析多。若用 SDK，直接采用 SDK 模型实例和消息类型，不再套自建 provider 接口。
- `ProviderState` 把 Gemini Interaction/cache 字段放进通用状态；改成按 provider 区分的少量状态，
  不要求 DeepSeek/Sub2API 存一堆无效 Gemini 字段。
- `src/app.ts` 682 行，同时混合注册、权限、输出、路由和定时任务；拆成少数功能模块，
  而不是为了拆文件添加 controller/service/use-case/port/adapter 层层传递。
- repository 当前确实封装 AI 状态事务，有存在价值；新 schema 的业务查询直接集中， 不再增加
  IRepository、generic repository 或额外 UnitOfWork。
- SHA-256
  媒体校验、事务、取消/进程超时、singleflight、会话互斥都有真实消费者，不应作为“抽象过多”一起删除。
  测试可以传入 fetch/runProcess/now 等少量函数，不必为每个函数造接口和实现类。

## 建议层级

`Telegram 功能与顺序编排 → 直接数据库查询 / SDK / 外部工具`

跨功能仅保留：

- 一次请求的薄 Interaction 对象：reply()、aiContext() 和消息身份关联。
- 一个 AI 执行入口：选择回复链、冻结请求、调用 SDK、保存结果、交付。
- 可见的事务与安全工具函数：解决可靠性，不发展成框架。

传统功能注入上下文不是模型工具调用，不能因为 SDK 有 tools/Agent 就改成模型自主执行传统功能。

## 接入方案

| 渠道                | 候选                            | 必须保留的区别                                                              |
| ------------------- | ------------------------------- | --------------------------------------------------------------------------- |
| 官方 Gemini         | Vercel Google provider          | generateContent/Interactions 分开；官方缓存/远端状态隔离                    |
| Sub2API             | 同一 Google provider 的独立实例 | 自定义 baseURL、独立凭证、3.7 Flash → tiered 映射；只走原生 generateContent |
| DeepSeek            | Vercel DeepSeek provider        | 图片能力按所选模型判断；维持当前最新 Go 的 active user image 行为           |
| Gemini 特殊管理接口 | 必要时少量 @google/genai 调用   | 显式缓存生命周期等不是统一 generateText 自动解决；禁止未触发时后台付费调用  |

官方 Google provider 支持自定义 baseURL/fetch、cachedContent、Interactions 与 provider metadata。
显式缓存创建示例仍使用 Google Gen AI SDK；不能把“传缓存名”等同于“管理缓存生命周期”。
依据：[Google provider](https://ai-sdk.dev/providers/ai-sdk-providers/google)。

DeepSeek provider 有自己的 provider options；模型名称能序列化不代表实验模型目前仍对账号开放。
依据：[DeepSeek provider](https://ai-sdk.dev/providers/ai-sdk-providers/deepseek)。

## 最新正式版本的离线验证

仅 /tmp 验证，没有修改 deno.json/deno.lock 或调用真实 provider。 npm registry 发布记录：

- ai 7.0.127
- @ai-sdk/google 4.0.87
- @ai-sdk/deepseek 3.0.58
- @google/genai 2.27.0（仅查询版本，未用它做 mock）

实际用 Deno 2.9.3 + 前三项 mock：

1. Sub2API 请求精确发往 `/antigravity/v1beta/models/gemini-3.7-flash-tiered:generateContent`。
2. Gemini 模拟响应的 thoughtSignature 在 response.messages 中保留，下一轮原生请求正确重放。
3. DeepSeek vision-exp 的图片序列化为 image_url/data URI；这是序列化测试，不是模型可用性验证。
4. 两者均正确返回 cached input 和 reasoning 用量：11 input、4 cached、3 text + 2 reasoning → 5
   output。 **不要再把 reasoning 加到 outputTokens，否则重复计算。**
5. 最新 generateText 默认不附带原始 response body；需 `include: { responseBody: true }`。
   持久化可重放 assistant messages（含 provider options）以及原始 provider payload，不能只存 text。
   如果需要 HTTP 原字节而非解析后的 JSON，可用窄 fetch hook 克隆响应保存，避免新建 transport 框架。
   依据：[generateText](https://ai-sdk.dev/docs/reference/ai-sdk-core/generate-text)。
6. Deno 可运行这些 mock，但 ai 根入口的间接 @vercel/oidc 导入读取 hostname： 需
   `--allow-sys=hostname`；无网络、无 allow-env 复跑也通过。现有 compile/test
   权限必须单独调整和验收， 不能直接认为 SDK 在旧权限集下即插即用。
7. 使用当前 file/image MIME 形式输入，避免沿用 deprecated image part。

没有真实 Gemini/DeepSeek/Sub2API、cache 创建、Google Interactions 远端链或生产迁移验收。
Interactions 的本地旧 steps payload 到 SDK
新消息格式仍须做导入/重放测试；不能只依据文档宣布完全兼容。

## 下一切片建议

确认 SDK 名称后，先做 SDK 适配的最小切片：

- 独立模型实例，直接 generateText；不创建新的 provider hierarchy。
- 将 /tmp mock 固化为无真实凭证的仓库测试，覆盖 raw payload、签名、usage 和 Sub2API 路由。
- 追加 Interactions、当前 DeepSeek 图片跟进/工具结果、缓存策略与重试用例。
- 指定有限重试与总 deadline，避免 SDK 默认重试再叠加当前 provider 的三次循环。
- 保留应用级 pending/generated/delivered 与一次计量事务；SDK 不负责 Telegram 交付。
- 再按新 schema/导入计划重组持久化与 aiContext；不要在一次提交里同时改 SDK、数据库和全 bot 架构。

SDK 净收益主要是手写协议减少和模型消息统一；不预先承诺整个仓库缩减比例。
