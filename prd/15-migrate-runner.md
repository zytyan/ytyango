# **Product Requirements Document**

## **产品名称：内置数据库迁移系统（migrate 子命令）**

## **版本：v1.0**

## **撰写日期：2025-12-24**


## **1. 背景（Background）**

* 当前 SQLite 数据库结构演进需要手工执行 SQL 或额外脚本，容易遗漏步骤、难以回滚。
* 主程序缺少启动时的版本校验，可能在 schema 不匹配时继续运行，导致读写异常。
* 需要一个可嵌入单一二进制的迁移能力，既支持 SQL 变更，也支持 Go 代码层的类型转换和数据修复。

---

## **2. 目标（Objectives）**

* 新增 `migrate` 子命令，支持升/降级到指定版本，并提供 DRY RUN 预览。
* 主程序启动时检查数据库版本，不匹配即 panic 并提示迁移。
* 迁移过程尽可能原子化，支持 SQL 与 Go 逻辑混合的迁移步骤。
* 仅依赖 SQLite，全部迁移定义随二进制发布，无外部 SQL 依赖。

---

## **3. 非目标（Out of Scope）**

* 不实现多数据库引擎（PostgreSQL 等）的迁移方案，后续按需扩展。
* 不做多节点分布式锁或租约控制，单节点内运行。
* 不提供自动定时迁移，仅手动或外部编排触发。

---

## **4. 用户角色（User Personas）**

### **开发者 / 维护者**

* 需要在发布前对数据库做升级或回滚。
* 需要在 CI 中以 DRY RUN 方式验证迁移脚本正确性。
* 希望迁移失败后数据库不处于中间态。

---

## **5. 用户故事（User Stories）**

1. **作为维护者，我可以执行 `ytyango migrate up --to 5 --db path.db`，将数据库升级到版本 5。**
2. **作为维护者，我可以使用 `--dry-run` 看到将执行的 SQL 和 Go 变更列表，确认后再正式执行。**
3. **作为开发者，我运行主程序时，如果数据库版本滞后，程序会 panic 并提示运行 migrate。**
4. **作为维护者，当迁移失败时，数据库保持在先前版本或自动回滚，不出现部分完成状态。**

---

## **6. 功能需求（Functional Requirements）**

| ID   | 描述 | 优先级 |
| ---- | ---- | ---- |
| FR-1 | 新增 `migrate` 子命令，支持 `up`、`down`、`to <version>`，目标版本可通过 flag 传入 | 高 |
| FR-2 | 支持 `--dry-run` 模式，仅打印将执行的 SQL/步骤，不对数据库写入 | 高 |
| FR-3 | 主程序启动时读取迁移元数据表（版本、dirty 状态），若与编译时预期版本不一致则 panic，提示运行 `migrate` | 高 |
| FR-4 | 迁移定义以连续版本号存放于 `migrate/` 目录，包含 Up/Down SQL 与可选 Go 回调，不依赖 sqlc | 高 |
| FR-5 | 默认在单事务内执行一个版本的迁移步骤，确保原子性；对 SQLite 不支持的 DDL 需给出分步安全策略并标记 dirty | 中 |
| FR-6 | 迁移步骤可包含 Go 数据处理（如类型转换、重建索引、填充默认值），并参与 DRY RUN 说明 | 中 |
| FR-7 | 迁移元数据表记录当前版本、目标版本、执行时间、dirty 标记与日志摘要，失败时可安全重试 | 中 |
| FR-8 | 命令行输出详细日志（执行 SQL、耗时、dry-run 展示），并返回非 0 退出码于失败 | 中 |
| FR-9 | 仅内置 SQLite 驱动，避免引入其他数据库依赖以控制二进制体积 | 低 |

---

## **7. 非功能需求（Non-functional Requirements）**

* 单一二进制：迁移定义与 SQL 均编译进可执行文件，无外部脚本或 .sql 文件。
* 可靠性：同一版本迁移失败时标记 dirty，下一次执行可检测到并先回滚/清理后重试；尽量使用事务保持原子性。
* 可维护性：迁移版本号、描述与变更点集中声明，提供模板/规范，避免重复代码。
* 可测试性：提供单元测试覆盖版本检测、dry-run 输出、成功/失败场景。
* 性能：迁移过程中避免长时间全表锁，必要时提供提示或分步策略。

---

## **8. 技术方案（Tech Design Summary）**

* 目录：新增 `migrate/`，包含迁移注册表、版本常量、执行器、CLI 入口。
* 版本管理：创建 `schema_migrations`（或同名表）记录 `version`、`dirty`、`applied_at`、`log`；版本号为递增整数。
* 启动校验：主程序读取元数据表，若不存在则初始化为版本 0；若与内置 `ExpectedSchemaVersion` 不一致则 panic，并输出迁移命令提示。
* 迁移定义：使用 Go 结构体切片维护每个版本的 Up/Down，Up/Down 可包含 SQL 字符串列表和可选 Go 函数；SQL 直接通过 `database/sql` 执行。
* 原子性策略：默认单事务执行一个版本的所有 Up/Down SQL；遇到 SQLite 不支持的多语句事务变更（如某些 `ALTER TABLE`）时，使用可回滚的临时表方案，并在 dirty 记录中注明。
* DRY RUN：执行器在 dry-run 时不写库，打印执行顺序、SQL、预估影响表，并标记将写入的版本变化。
* CLI：在 `cmd` 或主入口中新增 `migrate` 子命令，支持 `--db`（路径）、`--to`、`--step`、`--dry-run`、`--force-down` 等 flag；输出 JSON/文本日志。
* 扩展性：保留接口以按需引入其他数据库实现，但默认仅构建 SQLite，避免体积膨胀。

---

## **9. 数据结构（Data Models）**

`schema_migrations` 表示例：

```sql
CREATE TABLE schema_migrations (
  version INTEGER NOT NULL,
  dirty INTEGER NOT NULL DEFAULT 0,
  applied_at TEXT NOT NULL,
  log TEXT
);
CREATE UNIQUE INDEX idx_schema_migrations_version ON schema_migrations(version);
```

迁移定义结构（Go 草案）：

```go
type Step struct {
    SQL []string
    Go  func(ctx context.Context, db *sql.DB) error
}

type Migration struct {
    Version int
    Name    string
    Up      []Step
    Down    []Step
}
```

---

## **10. 里程碑（Milestones）**

| 时间 | 目标 |
| ---- | ---- |
| Day 1 | 完成 PRD 与迁移设计讨论，确定版本管理策略与目录结构 |
| Day 2 | 实现迁移执行器、元数据表与 dry-run 逻辑，完成基础测试 |
| Day 3 | 集成主程序版本校验与 `migrate` CLI，补充文档与更多测试 |

---

## **11. 使用说明（How to Use）**

* 升级到最新：`./ytyango migrate up --db ./data/main.db`
* 指定目标版本：`./ytyango migrate to --to 5 --db ./data/main.db`
* 回滚一步：`./ytyango migrate down --step 1 --db ./data/main.db`
* 预览：`./ytyango migrate up --to 5 --dry-run --db ./data/main.db`
* 主程序校验：运行主程序时若报版本不一致，按提示运行上述命令完成迁移。
