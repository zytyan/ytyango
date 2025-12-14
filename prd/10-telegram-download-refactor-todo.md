# Telegram 下载路径复用 TODO

- [x] 需求确认
  - [x] 完成 DownloadToMemory/Disk 统一封装的 PRD。

- [x] 下载逻辑统一
  - [x] 抽象 downloadToWriter，支持本地直读与远程下载。
  - [x] DownloadToMemory/DownloadToDisk 复用 downloadToWriter。
  - [x] DownloadToDisk 优先使用 DownloadToMemoryCached 的数据写入磁盘。

- [x] 清理与验证
  - [x] 保持 singleflight/LRU 语义不变，缓存淘汰继续清理文件。
  - [x] 运行 `gofmt` 与 `go test ./...`，记录结果。
