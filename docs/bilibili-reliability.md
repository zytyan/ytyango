# Bilibili 转换与下载修复（2026-10-15）

实现前刷新并检查 TS `9311011596d0f17fd852d50203efa4902b87182f`、Go 封存开发分支
`archive/go/20261013/codex/ai-db-v2` 的 `d395a28545701a5640d91b1ff7057bee236a3f1e`。 对照 Go
helpers/bili/converter.go 和 handlers/convertbili.go；短链无待清参数时不发送
转换结果是沿用的旧行为，本次修正为只要短链成功展开便视为需要转换。

线上日志确认下载失败于 findBilibiliCookies 的第一个相对路径 Deno.stat，抛出 NotCapable。目录迁移后
cwd 是仓库根目录，该相对候选不在编译允许读取的 build 下； 探测尚未到真正存在的
build/bili_cookies.json 就中止。修复优先使用唯一运行根下的
绝对账号文件路径，其他候选的不存在、权限拒绝均继续探测，不扩大编译读取权限。
实际验证生产服务用户可读取账号文件；临时下载验收使用副本，随后删除该凭据副本。

短链支持最多五次跳转、相对 Location、循环检测及整个链十秒期限，关闭中间响应体。 只访问
b23.tv/bili2233.cn，跳转目标限定 Bilibili 域名，不跟随任意外部目标。 HTTP/缺失跳转/JSON
错误返回经过清理的明确错误，不再将未解析短链交给下载工具。
下载入口同时规范化短链别名和普通视频链接，去掉跟踪参数、保留 p/t/start_progress，
非视频链接不提供下载按钮，也不启动视频下载。原消息仅在替换消息已发送后尝试删除，
删除失败不将成功转换标记为失败；转换失败保留原消息并记录/注入精简失败事实。

用户首个短链在服务器和用户端均失效，服务器返回 HTTP 200、JSON code=-404；
修复后直接说明上游错误，请用户重发或提供完整链接。补充短链可展开为
https://www.bilibili.com/video/BV1pmJy6mEre?p=1 ，AV/BV 转换与下载器元数据一致，
aid=117444001732258。

以 tgbotapi 用户实际调用现有 bbdown-go，下载得到 8790147 字节 MP4，ffprobe 验证时长约 252.992
秒，H.264 852×480 视频和 AAC 音频齐全。没有向真实 Telegram 发送验收消息。保持本地 API 的 file://
交付方式，不切换为 Web URL 或 multipart； 已有集成测试验证本地 API JSON 文件
URI、失败不落缓存、成功后缓存及音频路径。 独立编译 smoke 继续验证生产用户、受限权限、模拟
API、媒体及交付；另新增无参数短链、
相对跳转、失效短链、跨域/循环、受限文件探测与自动消息完整流程回归。

## Inline 上下文的 64 位 ID（2026-10-15）

本轮刷新分支后检查 TS `a798656e1565111cb3dd373dfc8b8d7c9346811d`，Go 封存开发分支仍为
`d395a28545701a5640d91b1ff7057bee236a3f1e`。

生产库只读检查发现 inline 上下文最大 UID 为 191315037158114041，超过 JavaScript 安全整数。 导入的 Go
int64 UID 使 SQLite 自增继续使用这个范围。原实现插入空上下文后，按 number 读取 last_insert_rowid()
越界；外层 catch 吞掉错误，最终回答通用的“无法转换”。这也解释了为什么 Telegram answerInlineQuery
成功而转换始终失败，以及库里出现未绑定的空上下文。

临时数据库用同一数量级的 UID 复现：修复前只返回 raw 结果，修复后得到两个精确 ID 的 AV/BV
结果。现在仅将该 ID 按 SQLite TEXT 读出并转为 bigint；回调十六进制解析、消息绑定、上下文
查找全程保持 bigint，验证正数及 int64 上限。不改生产数据库、自增序列或历史记录。

增加 inline.bilibili / inline.context 延迟阶段以及不含原查询内容的异常类型日志。Telegram 回答
移到转换异常捕获之外，避免把交付故障误报为转换故障或重复回答。回归覆盖大 ID 创建、回调编码、 via_bot
消息绑定、上下文读取/到期及 Telegram 回答失败只尝试一次。
