# **Product Requirements Document**

## **产品名称：Markdown Normalizer for Telegram Entities**

## **版本：v1.0**

## **撰写日期：2025-12-16**


## **1. 背景（Background）**

* AI 生成的 Markdown 直接发送到 Telegram 时，常出现不受支持的标记、偏移错误或转义缺失，导致消息展示异常。
* 目前业务缺少统一的 Markdown → Telegram text+Entity 规范化组件，人工修复耗时且易漏。
* 需要可复用、容错高的工具，用成熟 Markdown 解析器，从 AST 生成符合 Telegram UTF-16 code-point的 MessageEntity。

---

## **2. 目标（Objectives）**

* 输入：任意 AI 生成的 Markdown 文本。
* 输出：Telegram 可接受的 `text` 与 `[]MessageEntity`，覆盖粗体、斜体、下划线、删除线、剧透、code、pre、blockquote、expandable_blockquote、链接、mention、custom emoji 等。
* 对 Telegram 不支持的 Markdown 元素，按既定降级策略保持信息完整：
  * 图片转为链接文本；有序列表保留数字，无序列表用圆点；表格整体收拢为无语言标记的代码块；公式转为行内代码；残留特殊字符全部转义。
* 在多语言、多 emoji、代理/转发场景下保证 UTF-8 → UTF-16 偏移准确。
* 设计为可扩展的库/模块，便于后续增加 AST 节点映射或规则。

---

## **3. 非目标（Out of Scope）**

* 不承担 Telegram 发送逻辑，仅生成 text + entities。
* 不做 Markdown 渲染预览页面。
* 不在当前阶段处理文件/音视频等非文本媒体的上传流程。

---

## **4. 用户角色（User Personas）**

### **Bot 开发者**

* 希望一行调用得到合法 text+entities。
* 需要扩展点以覆盖新格式或业务标记。

### **内容审核/运营**

* 希望 AI 内容在 Telegram 展示稳定，无异常符号。
* 需要可用的日志或错误提示快速定位失败原因。

---

## **5. 用户故事（User Stories）**

1. 作为 Bot 开发者，我希望将 AI 返回的 Markdown 直接交给库处理，得到偏移正确的 text+entities。
2. 作为 Bot 开发者，我希望当出现不支持的元素（如图片、公式、表格）时，库能自动按规则降级而不是报错。
3. 作为 运营人员，我希望异常输入也能生成可发送文本，并给出可读的错误/警告信息以便回溯。
4. 作为 维护者，我希望能方便地增加新的 AST→Entity 映射规则并有单元测试覆盖。

---

## **6. 功能需求（Functional Requirements）**

### **6.1 Markdown 解析与 AST 构建**

| ID   | 描述                                                              | 优先级 |
| ---- | ----------------------------------------------------------------- | ---- |
| FR-1 | 采用成熟、可扩展、容错强的 Go Markdown 解析库（例如 goldmark/gomarkdown），禁止自建解析器。 | 高 |
| FR-2 | 支持从 AST 节点生成内部中间表示，便于映射到 Telegram 实体。                      | 高 |
| FR-3 | 解析时需保留原始文本与节点位置信息，保证 UTF-16 偏移可计算。                         | 高 |

### **6.2 Telegram Entity 生成**

| ID   | 描述                                                              | 优先级 |
| ---- | ----------------------------------------------------------------- | ---- |
| FR-4 | 输出结构包含 `text string` 与 `entities []MessageEntity`，实体字段对齐 Telegram Bot API。 | 高 |
| FR-5 | 支持 bold/italic/underline/strikethrough/spoiler/code/pre(含 language)/blockquote/expandable_blockquote/text_link/text_mention/custom_emoji。 | 高 |
| FR-6 | 所有 offset/length 按 UTF-16 code units 计算，必须覆盖多字节字符与 emoji。               | 高 |
| FR-7 | 在预/代码块内正确转义反引号和反斜杠；在链接、custom emoji URL 部分正确转义 ) 与 \。           | 高 |
| FR-8 | 输出 text 中的剩余特殊字符 `_ * [ ] ( ) ~ ` > # + - = | { } . !` 需按文档转义。              | 高 |

### **6.3 不支持元素的降级与保留**

| ID   | 描述                                                              | 优先级 |
| ---- | ----------------------------------------------------------------- | ---- |
| FR-9 | 图片：转换为可点击链接文本（显示原 alt 或占位符），不生成 photo/media 类型。                 | 高 |
| FR-10 | 有序列表保留数字，无序列表统一用圆点符号，整体包裹在不带语言的 ``` 代码块内。                 | 高 |
| FR-11 | 公式（行内/块）：转为行内代码实体，内容原样保留。                                      | 高 |
| FR-12 | 表格、任务列表等其他不支持结构：以原始行文本加入代码块或行内代码，保持可读性并全部转义。            | 中 |

### **6.4 错误处理与容错**

| ID   | 描述                                                              | 优先级 |
| ---- | ----------------------------------------------------------------- | ---- |
| FR-13 | 解析失败或节点不完整时，输出可发送的降级文本并返回错误/警告信息，不因单个节点导致整体失败。      | 高 |
| FR-14 | 提供可选的严格模式：遇到无法降级的节点返回错误，并指明位置。                               | 中 |
| FR-15 | 对外暴露调试选项（如启用 AST dump、实体预览），默认关闭。                                  | 低 |

### **6.5 接口与集成**

| ID   | 描述                                                              | 优先级 |
| ---- | ----------------------------------------------------------------- | ---- |
| FR-16 | 提供核心函数 `Normalize(markdown string) (text string, entities []MessageEntity, err error)`。 | 高 |
| FR-17 | 保持独立目录 `mdnormalizer`，可作为 helpers/ 可复用模块；提供基础单元测试样例。             | 高 |
| FR-18 | 保持与现有 Telegram Bot 发送逻辑解耦，仅依赖标准库和选定的 Markdown 库。                  | 高 |

---

## **7. 非功能需求（Non-functional Requirements）**

### **7.1 性能与资源**

* 单次转换 p95 < 30ms（1KB 文本基准）；
* 无明显内存泄漏或大对象逃逸；适配并发调用。

### **7.2 可靠性与可维护性**

* 关键路径单元测试覆盖 AST→Entity 映射、UTF-16 偏移、转义规则、降级策略。
* 代码使用 gofmt 格式化，通过 `go test ./...`。
* 出错场景提供可读错误信息；降级逻辑有注释说明。

### **7.3 可扩展性**

* 映射层可插拔：新增节点映射不影响核心偏移/转义逻辑。
* 支持配置开关（严格/宽松模式、是否保留原 Markdown），默认兼容宽松。

---

## **8. 技术方案（Tech Design Summary）**

* 语言：Go（单模块）。
* Markdown 解析：优先 goldmark（AST 扩展性、容错好），若需更换需保持 AST 访问与位置数据能力。
* Entity 生成：统一中间表示，集中处理 UTF-16 偏移与转义。
* 降级策略：在 AST 层识别 unsupported nodes，落到代码块/行内代码；列表聚合；图片转链接。
* 测试：样例覆盖多 emoji、混合语言、链接转义、代码块转义、列表降级、公式降级。

---

## **9. 数据结构（Data Models）**

```go
type MessageEntity struct {
    Type          string
    Offset        int // UTF-16 code units
    Length        int // UTF-16 code units
    URL           string
    User          *tele.User
    Language      string
    CustomEmojiID string
}

type NormalizedMessage struct {
    Text     string
    Entities []MessageEntity
    Warnings []string // 降级或转义提示
}
```

---

## **10. 里程碑（Milestones）**

| 时间 | 目标 |
| --- | --- |
| Week 1 | 选定 Markdown 库，完成 AST→中间表示与转义规则设计，输出设计文档/样例。 |
| Week 2 | 实现基础映射（常规文本、粗体/斜体/下划线/删除线/剧透、code、pre、链接、引用），通过核心单测。 |
| Week 3 | 完成降级策略（列表聚合、图片转链接、公式降级、表格处理），补充多语言/emoji 偏移单测。 |
| Week 4 | 增加严格模式与警告输出，完善日志/调试选项，准备上线说明。 |
