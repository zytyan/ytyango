# **Product Requirements Document**

## **产品名称：Inner HTTP 备份 Zstd 压缩**

## **版本：v1.0**

## **撰写日期：2025-12-24**


## **1. 背景（Background）**

* `/backupdb` 当前输出 zip(deflate)，压缩比与速度有限，备份体积偏大。
* 迁移/灾备希望获得更高压缩比与更快 CPU 利用率，减少存储与传输成本。
* 需要在保持原有 manifest、选择范围、token 校验的基础上切换至 zstd 压缩。

---

## **2. 目标（Objectives）**

* 输出格式改为 zstd 压缩包（如 `.tar.zst`），压缩比明显优于 deflate。
* 保留原有 `main.db` / `msg.db` 选择与 `manifest` 元信息。
* 下载接口兼容 curl 等工具，文件名体现新格式。

---

## **3. 非目标（Out of Scope）**

* 不支持多线程分片上传或断点续传。
* 不引入全量加密或签名。
* 不改变 token 验证与路由路径（仍为 `/backupdb`）。

---

## **4. 用户角色（User Personas）**

### **维护/运营人员**

* 需要更小的备份文件，节省带宽与磁盘。
* 需要命令行友好、向后兼容的下载流程。

---

## **5. 用户故事（User Stories）**

1. **作为维护者，我请求 `/backupdb` 并得到 `.tar.zst` 文件，大小显著小于原 zip。**
2. **作为维护者，我用 `tar --zstd` 或 `unzstd | tar` 即可解包得到 `main.db`、`msg.db` 和 `manifest.json`。**
3. **作为维护者，我可以继续通过 `db=main|msg|all`、token 校验使用该接口。**

---

## **6. 功能需求（Functional Requirements）**

| ID   | 描述 | 优先级 |
| ---- | ---- | ---- |
| FR-1 | `/backupdb` 默认返回 zstd 压缩包（建议 `.tar.zst`），含 manifest 与所选数据库文件 | 高 |
| FR-2 | 文件名含时间戳与新扩展名（例如 `backup-YYYYMMDD-HHMMSSZ.tar.zst`） | 高 |
| FR-3 | 继续支持 `db=main|msg|all` 选择范围；非法值返回 400 | 高 |
| FR-4 | Token 验证逻辑保持不变（header `X-Backup-Token` / query `token`） | 中 |
| FR-5 | 错误场景记录日志并返回 5xx/4xx，成功记录压缩后大小与耗时 | 中 |
| FR-6 | 提供简要使用说明（curl 下载与 tar 解包示例） | 中 |

---

## **7. 非功能需求（Non-functional Requirements）**

* 性能：单次备份耗时 < 30s；压缩内存占用 < 100MB。
* 可靠性：临时目录与中间文件在异常时清理；压缩失败时不生成半成品输出。
* 兼容性：保持 HTTP 路径与参数向后兼容；文档提示解压方式。

---

## **8. 技术方案（Tech Design Summary）**

* 依赖：引入 `github.com/klauspost/compress/zstd`（或 Go 标准库等同支持）以流式写入 zstd。
* 封装：将当前 zip 流改为 tar writer + zstd encoder，依次写入 `main.db`、`msg.db`、`manifest.json`；或直接 zstd 压缩 tar 数据。
* 响应头：`Content-Type: application/zstd`（或 `application/octet-stream`），`Content-Disposition` 使用 `.tar.zst`。
* Manifest：格式与字段保持不变。

---

## **9. 数据结构（Data Models）**

无新增数据表；`manifest.json` 结构不变。

---

## **10. 里程碑（Milestones）**

| 时间 | 目标 |
| ---- | ---- |
| Day 1 | PRD 评审与方案确认 |
| Day 2 | 完成 tar+zstd 实现与测试，更新使用文档 |
| Day 3 | 内网验证下载/解包，准备合并 |
