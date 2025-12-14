# MathParser 性能优化 TODO

- [x] 需求确认
  - [x] 完成 PRD 编写（性能目标、非目标、回归指引）。

- [x] 基准与测试
  - [x] 添加 FastCheck/算术/幂/阶乘/排列组合基准，记录初始基线。
  - [x] 扩充错误用例（取模零、阶乘非法、排列组合越界、结果过大）。

- [x] 优化实现
  - [x] 词法层优化：按需替换全角字符、无正则 FastCheck、数字快速解析、Token/RPN slice 复用。
  - [x] 求值层优化：预分配栈、重用 `big.Rat`/`big.Int`、并行基准验证。

- [ ] 后续跟进
  - [ ] 继续压缩 allocs/op（当前 BenchmarkEvaluateSimple ≈33 allocs/op，目标 ≤4 未完全达成）。
  - [ ] 如新增算子/大表达式支持，补充对应基准与 benchstat 对比记录。
