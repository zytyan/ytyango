# **Product Requirements Document**

## **产品名称：LRU Cache (lrusf) 测试覆盖**

## **版本：v1.0**

## **撰写日期：2025-12-18**


## **1. 背景（Background）**

lrusf 作为通用缓存组件，被多个业务缓存复用。
当前缺少单元测试，导致核心行为（LRU 驱逐、singleflight 合并、Range 遍历等）难以及时发现回归。

---

## **2. 目标（Objectives）**

* 为 lrusf 提供稳定的单元测试基线，覆盖核心 API 行为。
* 覆盖 LRU 驱逐与 onEvict 回调的正确性。
* 覆盖 singleflight 合并并发 fetch 的语义。
* 覆盖 Range/Remove/TryGet 的行为一致性。

---

## **3. 非目标（Out of Scope）**

* 不引入新的缓存特性或 API。
* 不修改业务侧缓存调用逻辑。
* 不进行性能基准测试。

---

## **4. 用户角色（User Personas）**

### **维护者/开发者**

* 需要可重复的测试确保缓存行为稳定。
* 需要在修改缓存实现时快速发现回归。

---

## **5. 用户故事（User Stories）**

1. **作为维护者，我希望验证 Get 在并发场景下只执行一次 fetch。**
2. **作为维护者，我希望验证超出容量时会驱逐最久未使用的 key。**
3. **作为维护者，我希望验证 Remove 后不会再命中缓存。**
4. **作为维护者，我希望验证 Range 遍历返回的值类型正确且完整。**

---

## **6. 功能需求（Functional Requirements）**

| ID   | 描述 | 优先级 |
| ---- | ---- | ---- |
| FR-1 | Get 命中时返回缓存值且不触发 fetch | 高 |
| FR-2 | Get 未命中时触发 fetch，并写入缓存 | 高 |
| FR-3 | 并发 Get 同一 key 只调用一次 fetch | 高 |
| FR-4 | 超出容量时按 LRU 驱逐旧 key，并调用 onEvict | 高 |
| FR-5 | Remove 后 TryGet 返回未命中 | 中 |
| FR-6 | Range 返回所有 key/value，值类型正确 | 中 |
| FR-7 | Add 覆盖已存在 key 时更新值并保持可命中 | 中 |

---

## **7. 非功能需求（Non-functional Requirements）**

* 测试稳定可重复，避免时间敏感/偶发失败。
* 测试应独立运行，不依赖外部服务。

---

## **8. 里程碑（Milestones）**

| 时间 | 目标 |
| ---- | ---- |
| Day 1 | 编写测试用例并通过 `go test ./...` |
