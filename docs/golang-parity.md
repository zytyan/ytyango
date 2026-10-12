# Go 基线同步记录

每个切片重新查询远端，不将以下 SHA 当作冻结基线。

## Bilibili 下载

检查基线：Go `codex/ai-db-v2` `d395a28545701a5640d91b1ff7057bee236a3f1e`； TS 前置提交
`a7803bf6616e225afc41bd471cdcd86d41de8c81`。

改用本机 `/usr/bin/bbdown-go`，沿用 Go 的账户 JSON、工作目录、HEVC 优先、 音频独立下载与 JSON
输出协议；编译权限同步。 复用 Go build 中的
`bili_cookies.json`，不读取或打印账户内容，不提交运行文件。
元信息失败不丢弃已经成功下载的文件；两个并发操作均等待完成，输出路径必须位于临时目录。

验收：参数与输出校验、音频交付、元信息失败、临时目录清理，以及现有下载 singleflight 回归。
真实下载不纳入默认测试。

后续待完成：TS 独立 schema/查询源/只读导入、AI-first 编排、工具、Cloudflare 审核、 AI
反应偏好/多模态、内部 HTTP/备份/诊断与运行验收。

待用户明确：未触发 AI 的相册 caption 是否允许保存七天。

## TS 数据库与只读导入

Go 基线仍为 `d395a28545701a5640d91b1ff7057bee236a3f1e`；TS 前置 `4ab727a`。 独立 TS V1、规范 SQL
与生成验证工作流、只读 Go V9–V11 导入。保持业务 ID， 逐表校验完整行摘要（含 BLOB）、Token、媒体
hash/size、外键和 integrity； CoC
属性、未使用的相册索引与远端状态归档。目标目录必须全新，失败不发布。
临时库回归验证源文件不变、投票聚合不重复、payload 无损和坏媒体拒绝发布。

## AI-first 编排与用户偏好

Go 基线仍为 `d395a28545701a5640d91b1ff7057bee236a3f1e`；TS 前置 `46db834`。 传统 handler 后执行 AI
检查；只有明确提及、`@new` 或回复已交付的 AI 回答触发。 `@last`、`@数字`
只控制明确触发后的路由，不独立触发。 沿用 Go 默认新会话、回复续接、`@last` 续接当前会话，跨 topic
引用拒绝续接。 显式传统事实按 chat/topic/实际回复关联保存，以 user 上下文提供，不进入 system。
已有事实阻止同一 update 重跑传统副作用；已生成 Run 重发不下载输入、不重调模型。 回复链中
request-only 上下文进入本次请求；窗口按 50 条裁剪且始终不超过 200，
裁剪后才加载媒体。请求快照在模型调用前保存，媒体保存 hash/size 引用。 `/ai_reaction on|off`
用户全局偏好缺省开启。`%CHAT_ID%` 提示词变量已补齐。

验收：非触发普通消息不进 AI 表，传统回复不触发，同消息 /roll + mention 先执行传统功能，
后续明确回复取回事实，事实不升为系统指令；SDK 交付失败重投和一次计量回归通过。

## Go 配置、图片审核与处理

Go `d395a28545701a5640d91b1ff7057bee236a3f1e`；TS 前置 `d9046bf`。 Cloudflare Clef-flash 直接
HTTP、0/2/4/6 与 anime 两项 choice 校验、JPEG 768px/128KiB 规范化；
失败不保存分数。安全二次元自动入库，每张相册图片分类且最多回复一次； `/score`
显示真实入库状态，只对入库图片提供投票。随机图库恢复评分区间与多种表达。 Go Cloudflare/OCR YAML
字段直接解析，支持独立 TS database/media 环境覆盖； 相对路径按配置目录解析。serve 在任何 writable
打开前只读确认 TS schema，拒绝误写 Go 库。 图片 codec 使用现有 `/usr/bin/ffmpeg`、`ffprobe`，模板是
Go 源码中的静态功能资源， 不是用户媒体；编译嵌入资源并仅增加必要执行/读取路径权限。

验收：70 项回归、真实本机 FFmpeg 鱼图/prpr/WebP→PNG/审核 JPEG smoke 通过。 实际 Go V11
只读导入演练：98,732 AI messages、46,595 Runs、1,023,120 统计桶， 9,412 个媒体、2,066,541,946
字节；完整行/payload 摘要、Token、媒体 hash/size、外键、integrity 通过。 实际演练报告保存在
/tmp，不提交数据库、媒体或运行文件。

## 传统工具与群设置

Go `d395a28545701a5640d91b1ff7057bee236a3f1e`；TS 前置 `5631f01`。 Fraction.js
有理数计算器保留排列组合、阶乘、百分数、整数除法、乘方、根号、绝对值和微小值科学计数；
限制表达式与结果规模。恢复汇率别名与有界缓存、Azure OCR/缓存/限速及长文字分段，头像 prpr 缓存、 静态
WebP→PNG、鱼图自动拼接；传统结果可显式注入 AI 回复链。恢复五项非 CoC 群开关，管理员校验。
统计文字长度仅统计 text，与 Go 相同。验收：73 项测试通过，包含数学边界与模拟外部 API、 同消息
/calc + mention 的执行顺序及事实注入。

## 内部接口与运维

Go `d395a28545701a5640d91b1ff7057bee236a3f1e`；TS 前置 `ba28b9e`。 恢复 loopback:4019 /
BOT_INNER_HTTP=OFF、全路由 token、非 loopback 强制鉴权、受限 JSON body、 Mars/Dio 统计、动态 runtime
日志等级、运行内存指标与 V8 CPU profile（替代 Go pprof 格式）。 `/backupdb` 使用 SQLite backup API
的独立一致性快照，校验完整媒体目录映射、大小和 SHA256， 快照对应的 DB/媒体副本/清单打包为
tar.gz；media=0 明确标记不完整 AI 数据集。 格式允许 breaking change，下载文件扩展名与 Content-Type
同步。超时、取消、临时文件清理与单次打包并发限制。 `media-gc` 默认 dry-run，grace-hours
必填；--apply 须在运行停止时执行，仅移除目录中未登记的旧对象，
不删除数据库历史或有登记的媒体。没有自动执行生产清理。76 项回归与本机真实 tar smoke 通过。

## 本地 Telegram 文件

Go `d395a28545701a5640d91b1ff7057bee236a3f1e`；TS 前置 `99296f8`。 实际 Go 配置 getMe/getFile
探测发现本地 Bot API 返回绝对路径；TS 原 HTTP 拼接会导致图片能力不可用。 现在本地 API 仅读取当前 Bot
的目录、检查 realpath 防止符号链接越界，并对本地/HTTP 输入进行 20 MiB
流式上限和取消检查；错误不包含带凭据的文件路径。编译增加实际 Bot API 数据根读取权限。 77
项回归通过，真实本地图片下载通过（未发送 Telegram 消息）。

## 多媒体准备与配置一致性

Go `d395a28545701a5640d91b1ff7057bee236a3f1e`；TS 前置 `bc40c80`。 沿用 Go SUB2API_BASE_URL
覆盖；动画贴纸仅发送 emoji/贴纸标记，不把 TGS 当 WebP。
不支持或超限媒体注明未上传；下载失败明确提示，本次不创建部分 Run、不调用模型。
排行/日报注入精简事实。78 项回归通过。Go 配置真实小请求：官方 Gemini 3.7、DeepSeek V4 成功； Sub2API
模型列表正常，但与 Go 相同的 tiered 生成路径返回 404。用户已确认保留配置并继续切换。
