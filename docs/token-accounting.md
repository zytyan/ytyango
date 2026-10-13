# Token 计数核对（2026-10-13）

只读核对线上 TS 数据、原 Go 库和固定 SDK 版本；不修改计数，不提交数据库、正文或原始响应。 检查基线
Go `d395a28545701a5640d91b1ff7057bee236a3f1e`、TS 前置 `510dcb6`。

## 连续对话输入偏小

真实 Interactions 请求存在 usage.total_input_tokens=161，而同一响应的 raw_prompt_token=1129；
另一轮为 165 对 1668。程序保存 161/165，与 API 上报及固定 Google SDK 转换一致。 远端链使用
previous_interaction_id，当前 HTTP input 只发送本轮增量；本地 input_message_count
则记录选中历史窗口， 二者不是同一个量，不能将消息条数当作本次完整传输或计费数量。

[官方 Interactions usage 定义](https://ai.google.dev/api/interactions-api)将 total_input_tokens
描述为 prompt/context Token；没有文档化
raw_prompt_token。实际低输入可能是远端链的增量统计或服务上报异常， 尚不能认定它就是完整计费
Token，也不能用未文档化字段擅自改写数据库用量。 缓存为零也不能证明远端完整历史未被使用。

## 输出不等于可见字数

真实响应：total_output_tokens=485、total_thought_tokens=624，SDK outputTokens=1109。
这包含推理，推理不再次相加；不可见内容和 tokenizer 编码也使 Token 与字数不一一对应。 原生
Gemini：candidatesTokenCount + thoughtsTokenCount；DeepSeek：completion_tokens 已包含 reasoning。
[官方 Token 说明](https://ai.google.dev/gemini-api/docs/tokens)和
[SDK generateText usage](https://ai-sdk.dev/docs/reference/ai-sdk-core/generate-text)分别说明 Token
与 usage 口径。

## 工具输入与缓存

真实原生响应 promptTokenCount=5210、toolUsePromptTokenCount=4194、正文=283、推理=305。 SDK 保存输入
9404、输出 588，合计 9992，与供应商 totalTokenCount 完全一致。 Go 只取
promptTokenCount，会漏掉这类工具输入；TS 不恢复这个漏计。 cachedInputTokens
是输入中的缓存子集，不能再次加到总输入上。

## 累计与重试

核对时切换后 94 个已交付请求的数值均能由响应 usage 和 SDK 转换解释，新建会话与请求汇总差异为零。
6309 个历史会话的累计值不等于现存请求之和；原 Go 源库有完全相同的差异，迁移保留原值，没有在 TS
中新增。 这证明差异早于迁移，不能仅凭差异把历史累计覆盖成现存 Run 的和；历史成因仍需进一步追溯。

生成成功只提交一次累计，交付失败后重发不再生成或计量。 瞬时错误重试只计最终成功响应的
usage；没有返回 usage 的失败尝试不能推断供应商是否收费。 成功空文本保留 payload
与用量，避免丢弃一个已成功计费的响应再生成。

## 展示调整

用量按钮改为整数 Token，区分接口上报输入、正文与推理、输入内缓存，并标明协议。 Interactions
原始输入参考值仅显示为未确认口径的诊断数据，不替换累计值、不宣称它是账单值。 旧记录缺少原始 usage
时仍显示已保存的总用量。Telegram 200 字符限制优先保留主要计数。
