# TS 数据库与 SQL 工作流

TS V1/V2 独立于 Go V9–V11：`ts_schema_versions` 是 TS 自己的 checksum/version 历史。 保留现有 AI Run
事务、媒体引用、统计及评分约束；相似表名用于减少无价值的重写， 不表示运行时能打开 Go 库。CoC
属性只进入 `legacy_coc_attrs` 归档。

规范源码为 `sql/migrations/*.sql`、`sql/queries.sql`。 修改 SQL 后执行 `deno task db:generate`，生成
Deno 编译可嵌入的 schema 与命名 SQL 常量。 生成器在临时内存库建 schema，并 prepare
每条命名查询；`deno task db:verify` 验证生成物没有漂移。 业务代码直接使用
`db.get/run/all(SQL.Name, ...)`，不增加查询框架，不编辑或复制 Go sqlc 产物。 既有 inline SQL
是债务；新增查询必须先进入规范 SQL 源码并检查参数、约束和事务边界。 已发布的 migration
源码不可改写；后续 schema 变化必须增加独立 TS migration。

## 离线导入

源库使用 SQLite read-only 连接和读取事务；不修改 schema、数据、migration checksum 或线上文件。 支持
Go canonical V9/V10/V11 与旧 TS V9；更旧库先用 Go 自带只读离线迁移工具转换副本。
目标输出目录必须不存在。导入先写新 staging 目录，验证成功后才发布整个目录。 保留
chat/message/session/run 标识，报告映射关系；业务表校验条数、完整行摘要、Token 汇总、 原始
assistant/provider BLOB、媒体 hash/size、外键与 integrity。 不复用未经验证的远端
provider-state；原状态及相册 caption 单独归档；文件索引另映射到 TS V2（Go 不含 topic，标记未知
-1），不隐式删除旧历史。 不重建统计、修正评分或重复执行生成来伪造一致性。

所有回归使用临时库与拦截 API。用户已授权复用 Go 配置，最终停止 Go 后只读导入并替换当前服务。
