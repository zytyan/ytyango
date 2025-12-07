# **Product Requirements Document Example**

## **产品名称：MarsBot Image Rating System 2.0**

## **版本：v1.0**

## **撰写日期：2025-12-07**


## **1. 背景（Background）**

随着用户量增加，MarsBot 的图片评分系统出现以下问题：

* 评分写入压力上升，现有 SQLite Prefix-Sum 方案偶有锁竞争。
* 用户评分界面在大图场景下加载迟缓。
* 没有对重复图片进行判重提示，用户体验不佳。

需要推出 **2.0 版本评分系统**，提升性能、体验与扩展性。

---

## **2. 目标（Objectives）**

* 将评分延迟从 **p95 150ms → p95 40ms**。
* 新增图片相似度检测，防止重复评分。
* 前端评分界面本地化缓存，降低加载时间。
* 允许未来扩展更多评分逻辑（如权重、时间衰减）。

---

## **3. 非目标（Out of Scope）**

* 不涉及视频评分。
* 不对历史数据进行清洗。
* 不支持跨群同步评分。

---

## **4. 用户角色（User Personas）**

### **普通用户**

* 每天浏览图片、简单评分。
* 关注速度和简洁体验。

### **群管理员**

* 希望看到用户评分趋势与优秀图片。
* 需要统计面板（WebApp）。

### **系统维护者**

* 希望降低 DB 锁竞争和系统负载。
* 希望可观测、可调优。

---

## **5. 用户故事（User Stories）**

1. **作为普通用户，我希望快速打开评分页面，不等待超过 1 秒。**
2. **作为普通用户，我给出评分时希望立即反馈成功。**
3. **作为管理员，我希望看到评分最高的前 100 张图片。**
4. **作为维护者，我希望可以查看评分队列的处理状态。**

---

## **6. 功能需求（Functional Requirements）**

### **6.1 图片评分（Rating）**

| ID   | 描述                       | 优先级 |
| ---- | ------------------------ | --- |
| FR-1 | 用户可对单张图片评分（1–5）          | 高   |
| FR-2 | 若用户重复评分，则更新原评分           | 高   |
| FR-3 | DB 中需实时更新 prefix-sum 计数组 | 高   |
| FR-4 | 评分后需返回新的平均分与参与人数         | 中   |

---

### **6.2 图片相似度判重（Similarity Dedup）**

| ID   | 描述                                       | 优先级 |
| ---- | ---------------------------------------- | --- |
| FR-5 | 插入图片时生成 pHash/dHash                      | 高   |
| FR-6 | 若相似度 < 10（Hamming Distance），提示用户该图片可能已存在 | 中   |
| FR-7 | 提供 API 查询重复项                             | 中   |

---

### **6.3 WebApp 前端评分界面**

| ID    | 描述                      | 优先级 |
| ----- | ----------------------- | --- |
| FR-8  | 图片在前端使用 localStorage 缓存 | 中   |
| FR-9  | 界面支持夜间模式自动适配            | 中   |
| FR-10 | 加载下一张图片不超过 300ms        | 高   |

---

### **6.4 管理与统计**

| ID    | 描述                      | 优先级 |
| ----- | ----------------------- | --- |
| FR-11 | Web 仪表盘展示评分分布           | 中   |
| FR-12 | 支持按时间查询评分记录             | 中   |
| FR-13 | 提供 API 导出评分历史（限制 10k 条） | 低   |

---

## **7. 非功能需求（Non-functional Requirements）**

### **7.1 性能（Performance）**

* API p95 < 50 ms
* DB 并发写入 TPS ≥ 800

### **7.2 稳定性（Reliability）**

* 系统崩溃不丢失评分（WAL 模式）
* 评分队列支持延迟恢复

### **7.3 安全（Security）**

* 用户评分请求必须来自合法 Telegram WebApp initData
* 图片 Hash 值不可被逆推出原图

---

## **8. 技术方案（Tech Design Summary）**

### **8.1 后端**

* Go + Gin / Ogen
* SQLite（WAL, STRICT, WITHOUT ROWID）
* Prefix-sum rating counters stored in separate table
* C 扩展提供高速 Hamming Distance（8 bytes 优化）

### **8.2 前端**

* Vue 3 + Vite
* Telegram WebApp SDK
* IndexedDB 缓存图片元数据

### **8.3 API**

* `/api/image/next`
* `/api/image/{id}/rate`
* `/api/image/{id}/similar`
* `/api/admin/stats`

---

## **9. 数据结构（Data Models）**

### **Image**

```sql
CREATE TABLE images (
    id INTEGER PRIMARY KEY,
    hash_p BLOB NOT NULL,
    hash_d BLOB NOT NULL,
    url TEXT NOT NULL
) STRICT;
```

### **Rating**

```sql
CREATE TABLE ratings (
    image_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    score INTEGER NOT NULL CHECK(score BETWEEN 1 AND 5),
    PRIMARY KEY(image_id, user_id)
) STRICT;
```

### **Prefix-Sum**

```sql
CREATE TABLE rating_stats (
    image_id INTEGER PRIMARY KEY,
    c1 INT DEFAULT 0,
    c2 INT DEFAULT 0,
    c3 INT DEFAULT 0,
    c4 INT DEFAULT 0,
    c5 INT DEFAULT 0
) STRICT;
```

---

## **10. 里程碑（Milestones）**

| 时间     | 目标            |
| ------ | ------------- |
| Week 1 | 数据结构、DB 迁移    |
| Week 2 | Rating API 完成 |
| Week 3 | 前端界面完成        |
| Week 4 | Dedup 模块上线    |
| Week 5 | 性能测试与修复       |
| Week 6 | 正式上线          |

---

## **11. 风险（Risks）与对策（Mitigations）**

| 风险            | 影响      | 对策               |
| ------------- | ------- | ---------------- |
| 图片 Hash 误判率   | 用户误以为重复 | 调高阈值、联动 ORB 特征匹配 |
| DB 锁竞争        | 性能下降    | 批处理写入、减少事务范围     |
| WebApp API 限频 | 加载慢     | 本地缓存 + 预加载       |

---

## **12. 验收标准（Acceptance Criteria）**

* 每个功能点都有自动化测试覆盖。
* 性能指标达标。
* 所有关键流程（评分、加载、判重）无 p99 错误。

---

