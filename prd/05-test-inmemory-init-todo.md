# Test In-Memory Init TODO

- [x] 现状确认
  - [x] 审查 `globalcfg_init_for_tests.go` 与 `globalcfg_init_test.go` 初始化顺序，记录现有问题（覆盖 config、跳过 schema）。
  - [x] 确认 build tags / `testing.Testing()` 行为，识别潜在双 init 风险。

- [x] 设计与实现
  - [x] 调整测试路径初始化，确保不会调用 `initConfig()`/生产配置，始终复用内存 Config。
  - [x] 在测试初始化中加载 `sql/schema_*.sql` 并准备 `Q` 与 `Msgs`，避免 `msgDb` 打开磁盘文件。
  - [x] 清理或合并重复 init 逻辑，保持生产 init 不受影响。

- [ ] 验证
  - [x] 运行 `go test ./...`，确认使用内存 DB；`helpers/bili`、`helpers/exchange` 用例及 `myhandlers` 端口监听在沙箱中失败，其余包通过。
  - [x] 复查日志/注释，确保后续贡献者能理解测试初始化分支。
