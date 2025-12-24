# **Product Requirements Document**

## **产品名称：Inner HTTP 数据库备份下载**

## **版本：v1.0**

## **撰写日期：2025-12-24**


## **1. 背景（Background）**

* 当前 main.db（`database-path`）与 msg.db（`msg-db-path`）均为 SQLite + WAL，备份需手工登录机器拷贝，易受锁竞争与未落盘数据影响。
* inner_http 只暴露基础调试接口，缺少快速、安全的数据保护路径，迁移/回滚前备份不便。
* 需要在不中断服务的情况下生成可直接下载的本地备份包，便于人工拉取或自动化脚本使用。

---

## **2. 目标（Objectives）**

* 提供一键 GET `/backupdb`，几秒内生成包含两个数据库快照的压缩包。
* 备份过程不阻塞正常写入，确保数据一致性与可恢复性。
* 备份包具备时间戳与元信息，便于审计和溯源。

---

## **3. 非目标（Out of Scope）**

* 不做定时/增量备份调度，也不上传对象存储。
* 不覆盖恢复流程与恢复校验。
* 不新增复杂权限体系，仅依赖 inner_http 监听地址及可选 token。

---

## **4. 用户角色（User Personas）**

### **维护/运营人员**

* 需要在迁移、上线前快速拉取当前数据。
* 需要最小化停机和权限暴露风险。

---

## **5. 用户故事（User Stories）**

1. **作为维护者，我通过 curl 访问 `/backupdb`，可以得到包含 main.db 和 msg.db 的 zip 文件。**
2. **作为维护者，我可以通过查询参数只备份 main 或 msg，以减小文件体积。**
3. **作为维护者，我希望从 HTTP 头或 manifest 中看到时间戳和源路径，方便标记备份。**
4. **作为维护者，当路径不存在或写入异常时，接口返回错误并记录日志。**

---

## **6. 功能需求（Functional Requirements）**

| ID   | 描述 | 优先级 |
| ---- | ---- | ---- |
| FR-1 | 提供 `GET /backupdb`，默认同时备份 main 与 msg 两个 SQLite 数据库 | 高 |
| FR-2 | 支持 `?db=main|msg|all` 选择备份范围，非法值返回 400 | 中 |
| FR-3 | 备份使用 SQLite 安全快照（backup API/VACUUM INTO），不中断正常写入 | 高 |
| FR-4 | 响应为 `application/zip` 流式下载，文件名含时间戳，Content-Disposition 便于保存 | 高 |
| FR-5 | 压缩包内包含 `main.db`、`msg.db`（按选择）以及 `manifest.json`（备份时间、源路径、大小） | 中 |
| FR-6 | 失败场景返回 4xx/5xx 并记录 error 日志；成功时记录耗时与文件大小 | 中 |
| FR-7 | 若配置可选 token（如 env `GOYTYAN_BACKUP_TOKEN`），需在 Header 或 Query 校验；未配置则直接允许本地访问 | 低 |

---

## **7. 非功能需求（Non-functional Requirements）**

* 性能：单次备份耗时 < 30s，峰值内存 < 50MB，临时文件自动清理。
* 可靠性：备份文件在写入完成并 fsync 后再开始下载；异常中止时清理临时文件。
* 安全性：默认监听 127.0.0.1:4019；日志不输出敏感 token；校验失败返回 401。
* 可观测性：日志记录备份来源 IP、数据库文件大小、耗时与失败阶段。

---

## **8. 技术方案（Tech Design Summary）**

* 路由：inner_http `buildHandler` 新增 `/backupdb` GET。
* 备份：通过 `database/sql` 获取 `*sql.DB` 后使用 sqlite3 Backup/VACUUM INTO 将 main/msg 复制到 `os.CreateTemp` 生成的快照文件，完成后打包 zip，边读边写响应。
* 输出：zip 内含 `main.db`、`msg.db`（按选择）与 `manifest.json`（时间戳、源路径、文件大小、选择范围）。
* 安全：支持可选环境变量 token（如 `GOYTYAN_BACKUP_TOKEN`），通过 Header `X-Backup-Token` 或 query `token` 校验；默认不强制。
* 清理：使用 `defer` 删除临时文件与临时目录，错误时中断响应。
* 配置依赖：读取 `config.DatabasePath`、`config.MsgDbPath`，复用 logger `inner-http`。

---

## **9. 数据结构（Data Models）**

`manifest.json` 示例：

```json
{
  "timestamp": "2025-12-24T12:00:00Z",
  "databases": [
    {"name": "main", "path": "/data/main.db", "size": 12345678},
    {"name": "msg", "path": "/data/msg.db", "size": 234567}
  ],
  "options": {"db": "all"}
}
```

---

## **10. 里程碑（Milestones）**

| 时间 | 目标 |
| ---- | ---- |
| Day 1 | 完成 PRD 与接口设计讨论 |
| Day 2 | 完成 handler 实现与测试（含错误场景） |
| Day 3 | 内网验证下载、更新 TODO、准备合并 |
