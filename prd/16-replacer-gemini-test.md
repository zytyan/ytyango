# **产品需求文档**

## **产品名称：Replacer 与 Gemini ChatName 对齐测试**

## **版本：v1.0**

## **撰写日期：2026-01-09**


## **1. 背景（Background）**

helpers/replacer 的 CHAT_NAME 逻辑应与 handlers/gemini_ai.go 中的 chat name 生成逻辑保持一致，以便模板替换结果符合 Gemini 会话的命名规则。
当前测试覆盖未包含仅有 FirstName 的场景，需要补齐。

---

## **2. 目标（Objectives）**

* 为 CHAT_NAME 增加与 gemini_ai.go 一致的边界测试场景。
* 保证测试使用最小依赖与稳定输入。

---

## **3. 非目标（Out of Scope）**

* 不修改现有聊天名生成逻辑。
* 不新增或删除模板变量。
* 不更改 handlers/gemini_ai.go 业务行为。

---

## **4. 用户角色（User Personas）**

### **维护者**

* 需要通过测试确保替换结果与 Gemini 会话命名一致。

---

## **5. 用户故事（User Stories）**

1. **作为维护者，我希望 CHAT_NAME 在仅有 FirstName 的情况下输出正确值。**

---

## **6. 功能需求（Functional Requirements）**

| ID   | 描述                                                         | 优先级 |
| ---- | ------------------------------------------------------------ | ------ |
| FR-1 | 覆盖 chat 仅含 FirstName 时的 CHAT_NAME 替换测试              | 高     |

---

## **7. 非功能需求（Non-functional Requirements）**

* 测试可重复运行，不依赖外部服务。

---

## **8. 技术方案（Tech Design Summary）**

* 在 helpers/replacer/replacer_test.go 中补充一条新的 CHAT_NAME 用例。

---

## **9. 里程碑（Milestones）**

| 时间 | 目标                  |
| ---- | --------------------- |
| Day 1 | PRD 与待办清单完成     |
| Day 1 | 新测试用例完成并通过   |

---

## **10. 验收标准（Acceptance Criteria）**

* 新增测试覆盖 chat 仅有 FirstName 的场景。
* `go test ./...` 通过（若外部依赖导致失败需注明）。

