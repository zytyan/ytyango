# HTTP Frontend Svelte TODO

- [x] 项目初始化与配置
  - [x] 在 `http/frontend` 使用 SvelteKit + `adapter-static` 初始化，默认 `prerender = true`，支持静态托管。
  - [x] 配置 `package.json` 脚本：`dev`、`build`、`preview`、`check`、`lint`、`gen:api`；本地化字体与 SVG 资产。
  - [x] 在 `app.html` 引入 Telegram 脚本 `<script src="https://telegram.org/js/telegram-web-app.js?59"></script>`，并提供缺省安全检测以兼容 SSG。
  - [x] 添加 `.env.example`（如 `VITE_API_BASE_URL`, `VITE_TG_AUTH`）与 README 运行说明。

- [x] OpenAPI 客户端生成
  - [x] 配置 `gen:api` 使用 `http/openapi.yaml` 生成 TS 客户端与类型（如 `openapi-typescript-codegen`），输出到 `src/lib/api` 并纳入版本控制。
  - [x] 封装 fetch 层，统一 base URL、`X-Telegram-Init-Data` 注入、错误包装与一次重试策略。

- [x] 主题与样式体系
  - [x] 默认支持 Telegram CSS 变量；在变量缺失时根据 `prefers-color-scheme` 使用内置亮/暗色主题。
  - [x] 为头像占位准备 8 组亮色 + 8 组暗色调色板，基于 userId 哈希选色，确保对比度。
  - [x] 定义全局 CSS 变量与布局（暗色/亮色），本地化字体与图标资源，避免外部 CDN。

- [x] UI 组件与页面
  - [x] 实现可复用消息卡片 Svelte 组件：头像（请求失败回退首字母占位）、昵称/用户名、时间、正文摘要、星标按钮、更多菜单占位。
  - [x] 实现搜索输入组件：带放大镜图标、回车触发、loading/禁用状态、空态提示。
  - [x] 实现结果列表页：分页或“加载更多”交互（映射 page/limit）、加载骨架、错误/空态提示。
  - [x] 头像拉取失败自动回退首字母圆形背景，使用哈希调色板；星标状态保存在前端（可用 localStorage）。

- [x] 数据与接口联调
  - [x] 打通 `/search` 请求与结果渲染，支持 tgauth header；处理 4xx/5xx 提示。
  - [x] 集成 `/users/info` 补全昵称/用户名（超时/错误使用占位名），避免阻塞主渲染。
  - [x] 集成 `/users/{userId}/avatar?tgauth=...`，失败回退占位；确保 SSR/SSG 不崩溃（客户端再取）。

- [ ] 测试与质量
  - [x] 配置 Vitest + @testing-library/svelte，至少覆盖搜索组件与消息卡片的渲染/状态切换。
  - [ ] 运行 `npm run check`、`npm run lint`、`npm test`，记录依赖/网络要求。
    - [ ] 当前 `npm test` 在 Vitest + Vite 7 组合下因 `__vite_ssr_exportName__` 运行时缺失报错（测试未执行），需进一步调整 Vitest/Vite 兼容性或跳过组件测试。

- [ ] 截图与验收
  - [ ] 生成与参考视觉一致的暗色界面截图（搜索结果状态），存放于 `http/frontend/static` 或 `docs`。
  - [ ] 更新 TODO 完成度并提交。
