# ytyanbot-ts

`ytyanbot-ts` 是独立的 TypeScript/Deno 重构，功能分析和迁移以当前最新 Go/TS 提交为准，
不再以历史冻结 commit 定义目标。当前实现 AI、视频/音频下载与 Bilibili 转换、NSFW 检测和图库评分、
群消息统计/日报，以及已上线功能的群配置。

后续迁移沿用本仓库：**只裁剪 CoC，保留其他能力，但允许数据库与交互 breaking change**。 新 TS
数据库独立设计，旧数据通过离线导入接收；不要求运行时复用 Go 库或逐字兼容旧命令/回复。 AI
上下文仍按回复链，传统功能先执行并通过 `aiContext(...)` 显式注入事实，随后才检查是否触发 AI；
不采集普通群聊历史，不在未触发时调用模型。 功能清单见
[features](docs/features.md)，架构、阶段计划及剩余问题见
[AI-first 计划](docs/ai-first-plan.md)，简化层级与 SDK 评估见
[架构分析](docs/architecture-sdk-analysis.md)。普通 `/roll` 已补齐，包含旧 `delme`
删除按钮兼容和离线测试。

**AI SDK 切片已实现，但新数据库/aiContext 编排仍待改造**：数据库仍直接使用 Go 的 canonical SQLite
V9，只接受 V9， 不能替换当前 Go V11 线上库。程序不会迁移或修复旧库，也不会修改 schema。 AI
媒体继续使用 `ai-media/sha256/<prefix>/<sha256>`。生产切换前必须停止 Go Bot；两个进程
不得同时轮询同一 token 或写同一数据库。

## 固定工具链

- Deno `2.9.3`
- grammY `1.45.1` 的 Web/Deno export
- grammY runner `2.0.3` 的官方 Deno entrypoint
- `@std/yaml 1.1.2`、`@std/assert 1.0.19`、`zod 4.4.3`
- Vercel `ai 7.0.127`、`@ai-sdk/google 4.0.87`、`@ai-sdk/deepseek 3.0.58`
- SQLite 使用 Deno `node:sqlite`，无 ORM、FFI 或原生扩展

本机安装的官方 `deno-x86_64-unknown-linux-gnu.zip` SHA-256 为：

```text
8101865641cbede56f08ad19c0a67a87df84bce127fee0d3e3e1f7467717ffa6
```

所有模块版本和远程模块内容均由 `deno.lock` 固定。runner 使用官方 Deno 源码入口，并通过 import map
把它的 grammY 依赖固定到 `1.45.1/web`；这样编译产物无需开放整个环境变量空间。

## 结构

```text
src/
  ai/          provider、会话/Run repository、媒体存储和业务服务
  download/    URL/Bilibili 处理、进程执行、缓存和 singleflight
  nsfw/        Azure Content Safety、图库和用户评分
  stats/       统计缓存、V9 聚合和日报
  db/          canonical V9 schema、初始化和严格检查
  telegram/    grammY typed API 与 sendRichMessage 窄适配
  shared/      当前共享类型与有界并发原语（单实现端口计划收拢）
```

AI 直接使用 SDK 模型实例与 `generateText`，不经自建 Provider 接口/类/注册 Map。 其他部分目前使用
Telegram、media、process、clock 端口。后续优先直接使用成熟库，
收拢无实际替换需求的单实现端口，不把这套结构作为目标架构。SQLite 事务内不进行网络请求或异步操作。

## 配置和运行

从示例开始建立测试配置：

```sh
mkdir -p run
cp config.example.yaml run/test-config.yaml
export YTYAN_CONFIG_FILE=/opt/tgbots/ytyanbot-ts/run/test-config.yaml
```

支持 Go 版 YAML 字段，以及 `YTYAN_BOT_TOKEN`、`GEMINI_API_KEY`、`SUB2API_API_KEY`、
`DEEPSEEK_API_KEY` 覆盖。Bot token、Telegram API URL、数据库和媒体路径改变后必须重启；
白名单、provider key、NSFW 和日志配置可热重载，无效配置保留旧快照。

模型选择同时保存 provider/model；相同 `gemini-3.7-flash` 可选官方或 Sub2API。 Sub2API 使用独立
`sub2api-key`/`sub2api-base-url`，仅走原生 antigravity generateContent，映射到
`gemini-3.7-flash-tiered`，不复用官方缓存或 Interactions。实验 DeepSeek 选项只表示能力目标，
不保证该模型当前仍向账号开放。

`gemini-explicit-cache: true` 的新官方会话使用原生生成；稳定上下文至少 4096 tokens、
最近稳定消息至少五分钟前时才尝试创建 900 秒缓存。复用检查内容/提示词/模型/tools 指纹；
命中后发送整个未缓存尾部，而不是仅发最后一条。已有 Interactions 会话继续使用原协议，
不把签名/工具步骤压成显示文字创建缓存。关闭显式缓存的新官方会话使用 SDK Interactions，
兼容性错误才回退原生路径。

直接调用现有供应商，不经过 Vercel AI Gateway，也不改变现有计费渠道。 SDK
自动重试关闭；每次生成总期限 180 秒，仅明确的缓存失效/远端链失效/接口不兼容有有限回退。 原字节
provider 响应、native assistant 与含 providerOptions 的 SDK messages 均持久化； 输出 token 已包含
reasoning，不再次相加。发送失败使用已存结果重发，不重新请求模型。

CLI：

```sh
deno task check
deno task test
deno task compile

./dist/ytyanbot-linux-x86_64 --version
./dist/ytyanbot-linux-x86_64 db-init --database /opt/tgbots/ytyanbot-ts/run/new.db
./dist/ytyanbot-linux-x86_64 db-check --database /path/to/v9.db --media /path/to/ai-media
./dist/ytyanbot-linux-x86_64 serve
```

`db-init` 拒绝覆盖已有文件。`db-check` 以 SQLite read-only 模式校验九条 migration、checksum、
必需表/索引/触发器、退役对象、`integrity_check`、外键和可选媒体引用。

编译产物只允许读取 Go build 目录、仓库 `run` 和 `/tmp`，只允许写仓库 `run` 和 `/tmp`，
只允许执行以下程序：

```text
/usr/local/bin/yt-dlp
/usr/local/bin/dotnet-tools/BBDown
/usr/bin/ffprobe
```

它没有 shell 或 FFI 权限。SDK 根入口的间接 OIDC 模块需要 hostname 权限，因此编译和测试仅补充
`--allow-sys=hostname`，不开放全部系统信息或环境变量。 `deploy/ytyanbot-ts-test.service`
仅用于独立测试 Bot，不会替换生产单元。

## 已验证

- `deno fmt --check`
- `deno lint`
- `deno check src/main.ts`
- 无网络/环境权限、仅 hostname 权限的 unit tests
- 仅授予文件/环境权限的 integration tests
- Linux x86_64 独立二进制 `--version`、`db-init`、`db-check` smoke
- 在无 Deno/npm 的 PATH 中连接本地假 Telegram API，runner 预热后空闲 RSS 约 62 MiB
- SIGINT 可停止 runner、关闭配置 watcher 并 flush 后以状态 0 退出

真实 Gemini、DeepSeek、Bilibili 和独立测试 Bot 验收需要显式提供测试凭证，默认测试不会访问这些服务。

本次 SDK 切片全仓合计 62 项（55 项 unit、7 项
integration）通过，含三供应商拦截、签名/工具结果/历史重放、 Vision
图片跟进、缓存指纹/尾部/失效、取消，以及生成后交付失败重发不重复调用/计量。
格式、lint、类型检查、独立编译及临时数据库 `db-init`/`db-check` smoke 均通过；未启动线上 polling，
未调用真实 provider，也未修改生产数据库；离线通过不等同于真实供应商或生产切换验收。

## 迁移待办，不是裁剪

OCR、计算器、汇率、prpr、鱼图、webp2png、备份接口、内部 HTTP、等价诊断、媒体 GC，
以及相册、反应偏好等能力仍待迁移。Sub2API SDK 路由及联合模型选择已接通并离线验收。
回复链/传统事实编排、上下文引用加载与媒体选择仍按 AI-first 计划重组，不能把 SDK 输入序列化验收
等同于整个回复链/多模态产品验收。新库与导入器按新模型设计，不要求复刻 Go V10/V11 表结构。
历史数据和配置通过映射迁移，不会因 breaking change 隐式删除；尚未迁移的群配置按钮仍返回
“本版本暂未上线”。

CoC 人物属性、检定和战斗模块明确不迁移，历史属性数据默认保留。独立 `/roll` 不属于这一裁剪范围。
已经退役的旧消息搜索和 AI memory 不重新引入。
