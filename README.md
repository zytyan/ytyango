# ytyanbot-ts

`ytyanbot-ts` 是冻结 Go 版本 `b6376aec837a45854944ed68a5561f5349fc442a` 的 TypeScript/Deno
核心迁移。当前实现 AI、视频/音频下载与 Bilibili 转换、NSFW 检测和图库评分、
群消息统计/日报，以及已上线功能的群配置。

数据库直接使用 Go 的 canonical SQLite V9，程序不会迁移或修复旧库，也不会修改 schema。 AI
媒体继续使用 `ai-media/sha256/<prefix>/<sha256>`。生产切换前必须停止 Go Bot；两个进程
不得同时轮询同一 token 或写同一数据库。

## 固定工具链

- Deno `2.9.3`
- grammY `1.45.1` 的 Web/Deno export
- grammY runner `2.0.3` 的官方 Deno entrypoint
- `@std/yaml 1.1.2`、`@std/assert 1.0.19`、`zod 4.4.3`
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
  shared/      五个公共端口及有界并发原语
```

业务层只通过 `TelegramPort`、`AiProvider`、`MediaStore`、`ProcessRunner` 和 `Clock` 接触外部
能力。SQLite 事务内不进行网络请求或异步操作。

## 配置和运行

从示例开始建立测试配置：

```sh
mkdir -p run
cp config.example.yaml run/test-config.yaml
export YTYAN_CONFIG_FILE=/opt/tgbots/ytyanbot-ts/run/test-config.yaml
```

支持 Go 版 YAML 字段，以及 `YTYAN_BOT_TOKEN`、`GEMINI_API_KEY`、 `DEEPSEEK_API_KEY` 覆盖。Bot
token、Telegram API URL、数据库和媒体路径改变后必须重启； 白名单、provider key、NSFW
和日志配置可热重载，无效配置保留旧快照。

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

它没有 shell 或 FFI 权限。`deploy/ytyanbot-ts-test.service` 仅用于独立测试 Bot，不会替换生产单元。

## 已验证

- `deno fmt --check`
- `deno lint`
- `deno check src/main.ts`
- 默认无权限的 unit tests
- 仅授予文件/环境权限的 integration tests
- Linux x86_64 独立二进制 `--version`、`db-init`、`db-check` smoke
- 在无 Deno/npm 的 PATH 中连接本地假 Telegram API，runner 预热后空闲 RSS 约 62 MiB
- SIGINT 可停止 runner、关闭配置 watcher 并 flush 后以状态 0 退出

真实 Gemini、DeepSeek、Bilibili 和独立测试 Bot 验收需要显式提供测试凭证，默认测试不会访问这些服务。

## 本期明确暂缓

OCR、计算器、汇率、骰子/战斗、图片生成和转换、旧消息搜索、备份接口、内部 HTTP、pprof、 媒体 GC
等功能未注册。数据库中对应表和配置字段不会被删除或重置；旧群配置按钮只返回 “本版本暂未上线”。
