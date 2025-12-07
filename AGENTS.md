# 目录结构
```
.
├── bothttp           # 当前的 http 后端入口与路由
├── globalcfg         # 全局配置项与公共常量
│   ├── h             # 帮助文件
│   └── q             # 数据库操作，部分自动生成
├── helpers           # 外部服务、工具适配与通用能力
│   ├── azure         # Azure 相关工具与封装
│   ├── bili          # 哔哩哔哩相关功能
│   ├── cocdice       # 掷骰/跑团工具
│   ├── exchange      # 汇率、兑换等换算
│   ├── imgproc       # 图片处理与生成
│   │   └── assets    # 图片处理相关静态资源
│   ├── mathparser    # 数学表达式解析
│   └── ytdlp         # 音视频下载与处理
├── myhandlers        # 业务处理器与路由逻辑
├── prd               # 产品需求文档（Product Requirements Documents）
├── scripts           # 开发辅助脚本与数据迁移工具
└── sql               # 数据库 schema 与插件
    └── plugins       # 插件相关 SQL

```

# 模型要求
1. 当Agent基于新代码需求运行时，总是从当前位置新建一个分支，若当前仍有未保存的文件，则立刻commit一次后新建分支。
2. 当有新需求时，**不要**立刻生成代码，而是在prd目录下生成新的基于markdown的产品需求文档。
prd的样例参考`prd/00-example-telegram-marsbot.md`。
3. **每次**生成prd后，都进行一次 `git commit`，在这之后可能有用户手动修改，但用户可能不会commit，修改后需要重新检查prd，并commit。
4. 用户修改prd后，查看相关修改，并根据这些修改提出相应建议。
5. 在prd基本确认后，为对应的prd生成一份markdown格式的todo清单，命名为 `prd-name-todo.md`。
6. 为便于review，每当完成阶段性任务时，应刷新todo列表并commit修改。
7. 若实际情况与todo列表产生差异，不要删除todo项目，而是在该项目下新增子项目。
8. 每个prd及todo清单都应该持久化保存，不应该删除。
9. 一切完成时，将修改同步到master分支，保留本分支的历史记录，只在master分支上新增一个commit节点。

# 开发说明
- Go 单模块项目，模块定义在 `go.mod`。
- 依赖变更后运行 `go mod tidy`。
- 使用 `gofmt -w` 格式化 Go 代码。
- 提交前运行 `go test ./...`，如因缺少依赖或外部服务导致失败，请在说明中注明。
- 提交信息保持简短，使用英文描述。
