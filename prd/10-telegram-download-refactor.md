# **Telegram 下载路径复用与流式封装 PRD**

## **产品名称：Download 统一流式封装**

## **版本：v0.1**

## **撰写日期：2025-12-14**

---

## **1. 背景（Background）**

当前 `DownloadToMemory` 与 `DownloadToDisk` 分别维护下载逻辑，存在重复代码；磁盘下载无法复用已缓存的内存数据，仍需二次请求。需要抽象通用的 `downloadToWriter`，统一下载流程、减少重复 IO，并让磁盘路径优先使用内存缓存的数据。

---

## **2. 目标（Objectives）**

- 提供共享的 `downloadToWriter`，`DownloadToMemory`/`DownloadToDisk` 复用它，避免重复实现。
- `DownloadToDisk` 优先使用 `DownloadToMemoryCached` 的数据写入磁盘，减少网络请求。
- 保持函数签名与返回值不变，调用方无需改动。

---

## **3. 非目标（Out of Scope）**

- 不调整现有 LRU 容量或 singleflight 行为。
- 不改动内存缓存策略与键规则。
- 不新增外部存储或持久化元数据。

---

## **4. 用户故事（User Stories）**

1. 作为维护者，我希望下载逻辑只有一处实现，避免重复修改。
2. 作为调用方，我调用 `DownloadToDisk` 时能复用内存缓存数据，不再重复下载。
3. 作为调用方，我的返回值与函数签名保持不变。

---

## **5. 功能需求（Functional Requirements）**

| ID   | 描述                                                                 | 优先级 |
| ---- | -------------------------------------------------------------------- | ---- |
| FR-1 | 抽象 `downloadToWriter`，同时支持本地路径直读与远程下载              | 高   |
| FR-2 | `DownloadToMemory`、`DownloadToDisk` 基于 `downloadToWriter` 实现     | 高   |
| FR-3 | `DownloadToDisk` 优先复用 `DownloadToMemoryCached` 数据，避免重复下载 | 高   |
| FR-4 | 保持现有 singleflight/LRU 语义与函数签名不变                         | 中   |

---

## **6. 技术方案（Tech Design Summary）**

- 提供 `downloadToWriter(bot, fileId, func(io.Writer) error)`：处理获取文件、判断本地路径、下载并写入 writer。
- `DownloadToMemoryCached` 维持现状；`DownloadToDisk` 先查内存缓存并写入稳定文件，再 fallback 到 `downloadToWriter`。
- 保留 disk LRU + eviction 删除文件、singleflight 合并请求；使用稳定文件名策略不变。

---

## **7. 里程碑（Milestones）**

| 时间  | 目标                                |
| ----- | --------------------------------- |
| Day 1 | 完成设计与实现，复用通用下载函数      |
| Day 2 | 自测 `gofmt`、`go test ./...`       |
| Day 3 | 反馈微调并上线                      |
