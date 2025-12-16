# Markdown Normalizer TODO

- [x] 需求确认
  - [x] 完成 11-mdnormalizer PRD 撰写并评审通过。

- [x] 解析与设计方案
  - [x] 确定 goldmark 为解析库，使用 walker 自建 UTF-16 偏移。
  - [x] 设计 builder + Option/NormalizedMessage，覆盖实体映射与降级策略。

- [x] 核心实现
  - [x] 实现 Normalize(markdown) 输出 text + []MessageEntity，处理 UTF-16 偏移；实体模式下文本不再转义。
  - [x] 实现降级策略（图片转链接、列表按圆点/序号直接输出、公式行内代码、表格等 fallback）。
  - [x] 支持严格/宽松模式与可选警告输出。

- [x] 转义与偏移验证
  - [x] 编写 UTF-8→UTF-16 偏移工具，覆盖多语言与 emoji。
  - [ ] 调整后确认：代码块保持原样、不额外转义；custom emoji URL 仍需验证。
    - [x] code/pre 与普通文本不再额外转义，实体偏移验证。
    - [ ] custom emoji URL 仍需验证。

- [x] 测试与质量
  - [x] 补充单元测试：常规实体映射、降级策略、混合 emoji/多语言、错误容错。
  - [x] 运行 `gofmt` 与 `go test ./...` 并记录结果。
