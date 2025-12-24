## TODO 清单：内置数据库迁移系统（migrate）

## A. 目录与配置基线

* [x] **A1. 目录初始化**
    * [x] 创建 `migrate/` 公共执行器骨架与 `main/`、`msg/` 子目录
    * [x] 定义版本常量 `ExpectedSchemaVersionMain` / `ExpectedSchemaVersionMsg`
    * [ ] 添加迁移模板与示例占位（空迁移 0→1）
* [x] **A2. 配置加载**
    * [x] 从 globalconfig 读取 `config.yaml` 的 `database-path`、`msg-db-path`
    * [x] CLI flag 覆盖配置路径，缺失时输出友好错误
    * [x] 裸 `migrate` 命令帮助输出，含默认路径来源说明

## B. CLI 子命令与选项

* [x] **B1. 子命令框架**
    * [x] 支持 `up` / `down` / `to` / `status` 子命令
    * [x] `--target main|msg|all` 选择数据库
    * [x] `--db` 覆盖单库路径
* [x] **B2. DRY RUN / 日志**
    * [x] `--dry-run` 打印将执行的 SQL/Go 步骤与目标版本变更
    * [x] 结构化日志（执行 SQL、耗时、版本、dirty 状态）
* [x] **B3. 内存演练**
    * [x] `--memory-run` 将目标库所有表复制到 `:memory:`，仅抽样行数据
    * [x] 行级抽样参数（`--sample-rate` 或 `--sample-rows`）及默认值
    * [x] 内存演练运行真实迁移流程，结果不落盘

## C. 迁移执行器与元数据

* [x] **C1. 元数据表**
    * [x] 为 main/msg 创建独立 `schema_migrations`（version, dirty, applied_at, log）
    * [x] 初始化逻辑：不存在则写入版本 0
* [x] **C2. 执行流程**
    * [x] Up/Down/To 统一调度，单版本默认单事务
    * [x] Go + SQL 步骤混合执行，事务内错误标记 dirty 并回滚
    * [ ] SQLite 特殊 DDL 分步安全策略（必要时记录 dirty 提示重试）
* [x] **C3. 并发/锁**
    * [x] 单机场景，无额外分布式锁；确认 SQLite 事务足够
    * [x] 迁移运行时的版本/dirty 状态检查与幂等保护

## D. 启动校验与错误处理

* [x] **D1. 主程序校验**
    * [x] 启动读取 main/msg 版本，缺表初始化 0
    * [x] 版本不匹配或 dirty 时 panic，提示命令示例
* [x] **D2. 错误与退出码**
    * [x] CLI 失败返回非 0，输出易读错误（含目标库路径/版本）
    * [x] 记录迁移摘要到 metadata log 字段

## E. 测试与文档

* [ ] **E1. 单元测试**
    * [ ] 版本检测与 panic 行为测试
    * [x] DRY RUN 输出验证
    * [x] 内存演练（全表结构 + 行级抽样）成功/失败场景
    * [ ] Up/Down 成功与 dirty 恢复测试
* [ ] **E2. 文档**
    * [ ] 使用说明（示例命令、默认路径说明、内存演练）
    * [ ] 迁移编写规范（命名、版本递增、Go 步骤注意事项）
    * [ ] 体积控制说明：仅 SQLite 驱动、无外部 .sql 依赖
