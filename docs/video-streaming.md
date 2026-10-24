# 视频流式播放

2026-10-24：修复 TS 视频发送遗漏 `supports_streaming` 的问题。 统一的
`GrammyTelegramGateway.sendVideo` 默认传入 `supports_streaming: true`， 覆盖本地 Bot API 文件
URI、远端 multipart 上传和 Telegram file ID 缓存发送。 保留原有
caption、回复身份、时长和尺寸参数，调用方仍可显式覆盖该默认值。
该标记声明视频支持流式播放，实际播放能力仍取决于视频格式及客户端；不改变媒体文件或重编码。

本次实现前已刷新远端，检查 TS `origin/codex/retain-features` 的
`84262235faff1f1137b1f6e0bce0f947eaaf08cc`、`origin/master` 的
`ac67c5f7651d3e2a4a2225ae4bd3f3a9f455e979` 和 Go 保留主线 `origin/archive/go/20261013/master` 的
`8f5de9b9161d64d813e85c68a95258f879aa0dd8`。 Go `globalcfg/h/video_helper.go` 原有
`SupportsStreaming: true`，此处恢复该发送能力。 回归拦截 Telegram
API，检查上述三种发送路径的实际请求参数，不发送真实视频。
