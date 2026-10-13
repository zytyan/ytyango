# 汇率来源与更新时间

2026-10-13 本次检查 TS `f9ba4ce59b48ea69107bc4da4c945010785e40ce`、封存 Go
`archive/go/20261013/codex/ai-db-v2` 的 `d395a28545701a5640d91b1ff7057bee236a3f1e`。

查询依次采用 fawaz CDN、同项目的 Cloudflare 备用地址、原 ExchangeRate-API。
网络/HTTP/数据格式失败或所需币种缺失时进入下一来源，每个来源仅一次，超时 10 秒。
不合并不同来源的汇率表，保持汇率与声明日期一致；各来源/基础币种分别缓存一小时，
串行首次加载避免并发重复请求。成功结果缓存，即使所查目标币种缺失也可用于其他目标。
全部来源失败时返回原有失败提示，不伪造汇率或更新时间。

输入支持 2–12 位字母数字代码，包括 USDT、TONCOIN、1INCH，保留旧三字母紧邻输入及 RMB/NTD/SKW
别名。纯数字代码、非有限金额和路径字符不接受。较长代码之间用空格分隔。

显示始终使用来源声明时间，不使用本地请求或缓存时间。旧接口的 time_last_update_unix 转换为
UTC+8。fawaz 只有 date，项目源代码采用 toISOString().substring(0,10) 生成 UTC 日期；
没有提供具体更新时刻，因此显示该 UTC 日对应的 UTC+8 范围，例如
`2026-10-13 08:00:00 至 2026-10-14 08:00:00 前（UTC+8，来源仅精确到日）`。
不会把日期补成虚构的精确更新时间。同币种换算无需 API，因此不显示更新时间。 明确注入的 exchange
事实包含相同日期说明；非触发消息不调用模型。

实测 USD/CNY、USDT/CNY、BTC/USD 使用 fawaz；USD/CLF、FOK/USD 使用旧接口补充，
且保留旧接口自身的更新时间。模拟验收另覆盖 CDN 网络失败、日期/数值无效、并发缓存、
缺失币种、跨年时区转换以及实际 Telegram 输出与事实持久化。

来源：[fawaz 项目](https://github.com/fawazahmed0/exchange-api)、
[UTC 日期生成代码](https://github.com/fawazahmed0/exchange-api/blob/main/currscript.js)、
[旧接口](https://www.exchangerate-api.com/docs/free)。
