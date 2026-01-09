# **产品需求文档**

## **产品名称：Gemini 系统提示词 Replacer 替换**

## **版本：v1.0**

## **撰写日期：2026-01-10**


## **1. 背景（Background）**

Gemini 系统提示词已经迁移为独立模板文件，但当前仍使用 fmt.Sprintf 占位符。
为了统一模板能力与测试覆盖，需要使用 helpers/replacer 的变量替换机制。

---

## **2. 目标（Objectives）**

* 使用 replacer 变量替换生成 Gemini 系统提示词。
* 模板变量覆盖时间、聊天类型/名称、机器人名称/用户名。
* 输出格式与现有提示词保持一致。

---

## **3. 非目标（Out of Scope）**

* 不引入 if/条件分支模板能力。
* 不改变提示词语义内容。
* 不调整 Gemini 业务流程。

---

## **4. 用户角色（User Personas）**

### **维护者**

* 需要统一模板替换机制，减少维护成本。

---

## **5. 用户故事（User Stories）**

1. **作为维护者，我希望系统提示词由 replacer 变量统一替换生成。**

---

## **6. 功能需求（Functional Requirements）**

| ID   | 描述                                                | 优先级 |
| ---- | --------------------------------------------------- | ------ |
| FR-1 | 模板使用 %VAR% 变量并由 replacer 替换               | 高     |
| FR-2 | 变量覆盖 DATETIME/CHAT_TYPE/CHAT_NAME/BOT_NAME/BOT_USERNAME | 高     |
| FR-3 | 生成的提示词与当前格式一致                          | 高     |

---

## **7. 非功能需求（Non-functional Requirements）**

* 测试可重复运行，不依赖外部服务。

---

## **8. 技术方案（Tech Design Summary）**

* 更新 handlers/gemini_sysprompt.txt 使用 %VAR% 变量。
* handlers/gemini_ai.go 使用 helpers/replacer.NewReplacer 生成提示词。
* 使用 ReplaceCtx 传入时间、聊天与机器人信息。

---

## **9. 里程碑（Milestones）**

| 时间 | 目标                     |
| ---- | ------------------------ |
| Day 1 | PRD 与待办清单完成        |
| Day 1 | 模板替换与测试完成        |

---

## **10. 验收标准（Acceptance Criteria）**

* Gemini 系统提示词通过 replacer 生成。
* 模板变量替换结果与当前输出一致。
* `go test ./...` 通过（若外部依赖导致失败需注明）。

