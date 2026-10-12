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
