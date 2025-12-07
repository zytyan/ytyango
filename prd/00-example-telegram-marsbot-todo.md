
##  TODO 清单

## **A. 数据结构与数据库层**

* [X] **A1. Schema 设计**

    * [X] 定稿 images 表结构
    * [X] 定稿 ratings 表结构
    * [X] 定稿 rating_stats（prefix-sum）表结构
    * [X] 确认所有字段类型（STRICT / WITHOUT ROWID）
    * [X] 编写 SQL migration 文档与注释

* [X] **A2. 索引与性能**

    * [X] 为 hash_p / hash_d 建索引
    * [X] 为评分表建立联合主键 image_id + user_id
    * [X] 分析 query plan 并记录优化说明
    * [X] DB schema 评审（Review）

* [X] **A3. 评分写入事务**

    * [X] 实现评分 insert/update 逻辑
    * [X] prefix-sum 增量更新（事务内）
    * [X] 配置 SQLite WAL
    * [X] 写入性能测试（目标 300–800 TPS）
    * [X] 编写评分写入流程文档

* [X] **A4. 重复图片 SQL 模块**

    * [X] 编写 pHash / dHash 相似度 SQL
    * [X] 阈值参数化（允许 <10 可配置）
    * [X] SQL 覆盖测试
    * [X] 模块技术说明文档

* [X] **A5. SQLite C 扩展**

    * [X] 实现 hamming_distance（8 字节优化）
    * [X] 实现任意长度 fallback
    * [X] so 加载失败降级处理
    * [X] Python vs C 输出一致性测试
    * [X] 模块技术文档

---

## **B. 后端 API（Go / Ogen / Gin）**

* [ ] **B1. 公共模块**

    * [ ] Telegram initData 校验中间件
    * [ ] 统一 API 错误结构
    * [ ] zap 日志与指标埋点
    * [ ] Prometheus metrics 暴露
    * [ ] 公共模块文档 + API 审查

* [ ] **B2. Rating 主流程 API**

    * [ ] 实现 `/api/image/{id}`（获取图片详情）
    * [ ] 实现 `/api/image/{id}/rate`（写入评分）
    * [ ] 实现 `/api/image/next`（获取下一张）
    * [ ] Score 更新与并发单元测试
    * [ ] Postman/Bruno 测试接口整理
    * [ ] 模块文档 + Review

* [ ] **B3. 图片相似度 API**

    * [ ] 实现 `/api/image/{id}/similar`
    * [ ] 阈值配置化
    * [ ] 相似度 API 返回结构定义
    * [ ] C 扩展错误降级机制
    * [ ] 误判率单元测试
    * [ ] 模块文档 + 技术审查

* [ ] **B4. 管理员统计 API**

    * [ ] top-N `/api/admin/stats/top`
    * [ ] 分布 `/api/admin/stats/distribution`
    * [ ] 导出 `/api/admin/export`
    * [ ] 设计响应结构
    * [ ] 大数据量测试
    * [ ] 模块文档 + Review

* [ ] **B5. 后端集成测试**

    * [ ] 全链路评分测试（next → rate → stats）
    * [ ] 并发读写场景测试
    * [ ] Dedup 精度与距离分布验证
    * [ ] Ogen TS 客户端生成验证
    * [ ] 提交后端整体验收说明

---

## **C. 图片相似度模块（Hashing / Dedup）**

* [ ] **C1. Hash 生成**

    * [ ] 实现 pHash
    * [ ] 实现 dHash
    * [ ] ORB 后续扩展（可选）
    * [ ] 图片上传流水线：自动生成 hash
    * [ ] Hash 模块文档 + Review

* [ ] **C2. Dedup 逻辑**

    * [ ] Hamming 距离阈值匹配
    * [ ] 多 hash 组合策略
    * [ ] 批量误判率统计脚本
    * [ ] 模糊命中 UI 提示文本
    * [ ] Dedup 流程文档 + 审查

* [ ] **C3. 性能验证**

    * [ ] 每秒 hash 生成量测试
    * [ ] C 扩展 vs Python 性能对比
    * [ ] 1k/10k 图片压测
    * [ ] 写性能结果报告

---

## **D. 前端 WebApp（Vue 3 + Telegram WebApp）**

* [ ] **D1. 基础结构**

    * [ ] Vite + Vue + TS 基础项目
    * [ ] Telegram WebApp SDK 接入
    * [ ] 夜间模式适配（媒体查询 + Telegram theme）
    * [ ] 全局错误提示
    * [ ] 模块结构文档

* [ ] **D2. 用户评分界面**

    * [ ] 图片展示组件
    * [ ] 评分组件（点按/滑动）
    * [ ] 提前预加载下一张
    * [ ] localStorage 缓存图片元数据
    * [ ] 首屏加载优化（FCP < 1s）
    * [ ] UI 行为测试
    * [ ] 评分页面文档 + UI Review

* [ ] **D3. 管理后台**

    * [ ] Top 图片页（分页）
    * [ ] 评分分布图（echarts/recharts）
    * [ ] 时间区间筛选器
    * [ ] 导出按钮（连接后端 API）
    * [ ] 管理页 E2E 测试
    * [ ] 后台文档 + Review

* [ ] **D4. 前端集成验证**

    * [ ] Ogen TS 客户端接口一致性检查
    * [ ] 性能测试（加载速度、缓存命中）
    * [ ] API 错误处理验证
    * [ ] 整体验收文档

---

## **E. 可观测性（Observability）**

* [ ] **E1. Metrics 接口**

    * [ ] rating_write_latency
    * [ ] image_next_latency
    * [ ] dedup_query_count
    * [ ] 指标文档

* [ ] **E2. 日志**

    * [ ] 结构化 API 请求日志
    * [ ] DB 慢查询日志
    * [ ] dedup 命中与误判日志
    * [ ] 日志策略文档 + Review

* [ ] **E3. Grafana Dashboard**

    * [ ] 评分趋势图
    * [ ] DB 负载面板
    * [ ] API 性能面板
    * [ ] Dashboard 文档

* [ ] **E4. 压测**

    * [ ] API p95/p99 压测
    * [ ] SQLite WAL 性能压测
    * [ ] Dedup 查询压测
    * [ ] 输出压测报告

---

## **F. 部署与运维（Ops）**

* [ ] **F1. systemd 服务**

    * [ ] Protect 系列配置
    * [ ] HOME 目录位置规划
    * [ ] ReadWritePaths 最小化
    * [ ] systemd 文档

* [ ] **F2. 构建与部署**

    * [ ] CI/CD pipeline
    * [ ] Go 构建（含 CGO for C extension）
    * [ ] 前端构建（Vite）
    * [ ] 构建产物文档

* [ ] **F3. 上线策略**

    * [ ] 灰度发布（5% → 25% → 100%）
    * [ ] 监控阈值设置
    * [ ] 回滚策略
    * [ ] 最终上线验收文档

---

## **G. 验收（Acceptance）**

* [ ] 各模块自测报告
* [ ] 全链路 E2E 测试报告
* [ ] 性能指标达标验证
* [ ] 安全检查（initData、限频、接口权限）
* [ ] PRD 完成度检查清单
