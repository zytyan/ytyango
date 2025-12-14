# Telegram 文件下载磁盘缓存 TODO

- [x] 需求确认
  - [x] 完成 DownloadToDisk 缓存与 singleflight 的 PRD。

- [ ] 缓存与并发
  - [x] 为 DownloadToDisk 增加基于文件 ID 的 LRU 缓存，支持 singleflight 合并。
  - [x] 同一文件使用稳定文件名/路径。

- [ ] 清理与回收
  - [x] 在缓存淘汰时删除对应磁盘文件，避免残留。

- [ ] 验证
  - [x] 运行 `gofmt` 与 `go test ./...`，记录结果（均通过）。
  - [x] 根据测试结果调整或补充说明。
