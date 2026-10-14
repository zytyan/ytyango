# Gemini 4 Argon 接入（2026-10-14）

本次刷新后检查 TS `01fcf496540944551c8bacc085fd99ea921fc196`，封存 Go
`archive/go/20261013/codex/ai-db-v2` 的 `d395a28545701a5640d91b1ff7057bee236a3f1e`。

官方模型 ID 为 `gemini-4-argon`。当前官方账号的模型列表已返回该 ID；Sub2API 列表未返回任何
Argon，因此本次只增加官方渠道，不猜测代理的别名或 tiered 映射。 模型菜单末尾追加选项，原 aimodel2
索引不变，不更改现有聊天或会话默认模型。

继续通过 createAiGenerator/Vercel AI SDK 调用，官方优先使用 Interactions，保留既有
有界协议兼容回退和续聊事务。两种协议都显式采用官方推荐 medium 思考级别。 Argon 不支持显式缓存，故
countTokens/cachedContents 创建与引用均跳过；即使聊天 缓存开关启用，Argon
新会话也不会被该开关强制切到原生缓存路径。其他模型缓存策略保留。 隐式缓存命中按供应商返回的 cached
input 统计。输出 Token 已含思考，不再累加。

本次接入为同步普通生成，保留默认输出上限，不开启百万 Token 长输出、background
轮询或新增工具。输入多模态继续沿用现有 Gemini 能力；不将只有模型名称的接入认作
全部多模态已实测。保留回复链、原始响应、SDK/native payload、供应商状态及一次计量。

回归覆盖菜单索引、两轮 Interactions 增量续聊、medium 请求字段、原生兼容回退、
显式缓存开关及旧缓存状态不触发不支持的 API。真实官方两轮短请求验收单独记录在部署报告。

依据：[模型说明](https://ai.google.dev/gemini-api/docs/models/gemini-4-argon)、
[迁移与缓存说明](https://ai.google.dev/gemini-api/docs/latest-model)。
