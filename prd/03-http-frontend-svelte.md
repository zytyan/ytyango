# **Product Requirements Document — HTTP Frontend (Svelte SSG)**

## **产品名称：HTTP Web 前端（SvelteKit + SSG）**

## **版本：v1.0**

## **撰写日期：2025-12-07**


## **1. 背景（Background）**

后端已基于 `http/openapi.yaml` 和 ogen 输出搜索、用户信息、头像等接口，当前缺少对应的 Web 端展示。需要在 `http/frontend` 下新增 Svelte 前端，使用 SSG 预渲染主干页面，前端请求代码由 OpenAPI 驱动生成，减少手写样板。UI 需对标提供的暗色截图：顶部搜索框、卡片化结果列表、显眼的用户头像占位、收藏星标与更多操作。

---

## **2. 目标（Objectives）**

* 新建 `http/frontend` SvelteKit 应用，默认启用 SSG/预渲染，兼容静态托管。
* 请求与类型通过 `http/openapi.yaml` 自动生成（TypeScript client + schema types），避免重复定义。
* 实现“搜索/结果列表”核心页面，界面风格对标截图，支持星标、更多菜单等交互位。
* 支持头像拉取与占位渲染，头像接口与搜索接口均使用统一的 `tgauth`/header 鉴权。
* 交付可复用的构建脚本、代码生成脚本、基本检查/测试命令和示例截图。

---

## **3. 非目标（Out of Scope）**

* 不实现除搜索结果页外的其他业务页面（如设置、登录注册等）。
* 不重新定义或修改后端 OpenAPI；前端仅消费现有 schema。
* 不上线生产级国际化、多主题切换；保持单一暗色主题。
* 不接入复杂状态管理（如 redux/store）；首版以轻量 fetch 与本地 state 为主。

---

## **4. 用户角色（User Personas）**

### **终端使用者（运营/搜索用户）**

* 需要通过关键字快速检索消息，查看时间、正文、来源用户信息。
* 希望界面加载快、结构简洁，并可标记重点内容（星标）。

### **前端/全栈开发者**

* 需要类型安全的 API 客户端，减少维护成本。
* 需要 SSG 产物便于部署到静态环境或 CDN，同时支持运行时请求。

### **QA / 验收人员**

* 需要可复现的截图、稳定的本地启动与构建命令，便于对比 UI 与功能。

---

## **5. 用户故事（User Stories）**

1. 作为运营，我在页面顶部输入关键词，按回车即可看到按时间排序的搜索结果卡片，每张卡片展示发送者与时间。
2. 作为运营，我在结果卡片上点击星标按钮，将条目标记为重点，再次点击可取消。
3. 作为用户，我在网络较慢时看到加载骨架或状态提示，不会出现空白屏。
4. 作为用户，我在头像获取失败时自动看到彩色字母占位，而非破损图片。
5. 作为开发者，我运行 `npm run gen:api` 从 `http/openapi.yaml` 生成 TS 客户端，避免手写请求定义。
6. 作为运维，我需要 SSG 产物可直接部署到静态托管，并在部署前有 `npm run check`、`npm run lint` 等基础校验。

---

## **6. 功能需求（Functional Requirements）**

### **6.1 项目结构与生成**

| ID   | 描述                                                       | 优先级 |
| ---- | -------------------------------------------------------- | --- |
| FR-1 | 在 `http/frontend` 创建 SvelteKit 应用，默认启用 `prerender`（SSG）。 | 高  |
| FR-2 | 配置 `package.json` 脚本：`dev`、`build`、`preview`、`check`、`lint`、`gen:api`。 | 高  |
| FR-3 | `gen:api` 使用 `http/openapi.yaml` 生成类型与客户端；生成物存放于 `src/lib/api`（或类似目录），可被 git 跟踪。 | 高  |
| FR-4 | README/脚本说明如何在仓库根或 `http/frontend` 目录下运行前端与生成代码。 | 中  |

### **6.2 UI/UX 与交互**

| ID   | 描述                                                                                   | 优先级 |
| ---- | ------------------------------------------------------------------------------------ | --- |
| FR-5 | 页面默认暗色主题，整体布局参考截图：顶部搜索输入框，下方为卡片列表。                                | 高  |
| FR-6 | 搜索框支持输入后回车触发，显示 loading 状态；空搜索时展示引导文案。                                  | 高  |
| FR-7 | 结果卡片包含：圆形彩色头像（首字母）、昵称/用户名、时间（相对或格式化）、正文摘要、多行消息对齐。             | 高  |
| FR-8 | 卡片右上角提供星标按钮，支持切换状态；右侧提供“更多”菜单占位（无需真实功能，可下拉弹出选项占位）。             | 中  |
| FR-9 | 支持结果分页/“加载更多”或无限滚动（至少一种）；分页参数映射后端 `page/limit`。                         | 中  |
| FR-10 | 在请求失败/空结果时给出错误/空态提示；网络缓慢时展示骨架或 shimmer 效果。                            | 中  |
| FR-11 | 提供示例截图（与需求截图一致风格）存放于 `http/frontend/static` 或 `docs` 供验收。                   | 中  |

### **6.3 数据与鉴权**

| ID   | 描述                                                                          | 优先级 |
| ---- | --------------------------------------------------------------------------- | --- |
| FR-12 | 搜索接口调用 `/search`，附带 `X-Telegram-Init-Data`（或约定 header）以满足后端 `TgAuth`。 | 高  |
| FR-13 | 批量用户信息接口 `/users/info` 可用于补全昵称/用户名；超时或错误时使用占位名称。                | 中  |
| FR-14 | 头像接口 `/users/{userId}/avatar?tgauth=...`，失败时退回到首字母圆形背景。背景颜色基于用户ID进行随机选择，为亮色与暗色分别配置8个颜色。                      | 高  |
| FR-15 | 请求客户端自动处理 base URL（支持环境变量配置）、错误包装与重试策略（简单重试 1 次即可）。               | 中  |

### **6.4 性能与部署**

| ID   | 描述                                                                 | 优先级 |
| ---- | ------------------------------------------------------------------ | --- |
| FR-16 | 默认 SSG 产物输出（`npm run build`）可直接用于静态托管（如 Nginx/CDN）；保留 `preview`。 | 高  |
| FR-17 | 静态资源（字体/图标）本地化存放；避免加载外部 CDN 以便离线验收。                      | 中  |
| FR-18 | 关键资源 gzip/压缩开启（依赖托管环境即可），构建产物大小可控。                       | 低  |

### **6.5 可测试性**

| ID   | 描述                                                                      | 优先级 |
| ---- | ----------------------------------------------------------------------- | --- |
| FR-19 | 提供基础单元测试或组件测试（如 Vitest + Testing Library）覆盖搜索组件的渲染与状态切换。 | 中  |
| FR-20 | CI/本地可运行 `npm run check`、`npm run lint`，确保类型/样式一致。              | 高  |
| FR-21 | 生成客户端的流程需在测试前可重复执行；缺少后端时可通过 mock 数据驱动主要组件渲染。        | 中  |

---

## **7. 非功能需求（Non-functional Requirements）**

* **可维护性**：OpenAPI 变更只需重新运行 `gen:api`；客户端封装为单点模块，避免散落请求定义。
* **可用性**：暗色主题下对比度满足基本可读性（WCAG AA 对文本/按钮）；键盘可访问搜索框与星标按钮。
* **性能**：首屏渲染依赖 SSG，交互请求仅加载必要数据；分页请求应避免一次性加载过多数据。
* **部署**：构建输出与文档明确，支持 docker/nginx 静态托管或 SvelteKit `adapter-static`。

---

## **8. 技术方案（Tech Design Summary）**

* 使用 SvelteKit + `adapter-static` 实现 SSG；大部分页面预渲染，搜索结果通过 CSR 触发数据获取。
* 通过 `openapi-typescript-codegen`（或同类工具）从 `http/openapi.yaml` 生成 TS 客户端与类型，输出至 `src/lib/api`。
* 封装 fetch 层，注入 base URL、认证 header（`X-Telegram-Init-Data`）与错误处理。
* UI 基于两种模式（亮色、暗色）、卡片化设计，使用自定义 CSS 变量；图标可用本地 SVG，字体使用本地化无衬线字体。
* 头像拉取失败时回退到彩色字母背景；星标状态保存在前端（本地存储）即可，暂不落库。
* 提供示例配置 `.env.example`（如 `VITE_API_BASE_URL`, `VITE_TG_AUTH`），用于本地开发与静态托管。
* 测试使用 Vitest + @testing-library/svelte 覆盖关键组件；提供简单 mock 数据驱动渲染。
HTML中必须包含telegram提供的脚本
```html
<script src="https://telegram.org/js/telegram-web-app.js?59"></script>
```

## 8.1 消息卡片
为消息结果制作可复用的消息卡片，以svelte文件方式提供。

## 8.2 背景
由于telegram提供了css变量，应优先使用telegram的css变量。
### ThemeParams

Mini Apps can [adjust the appearance](https://core.telegram.org/bots/webapps#color-schemes) of the interface to match the Telegram user's app in real time. This object contains the user's current theme settings:

| Field                     | Type   | Description                                                  |
| :------------------------ | :----- | :----------------------------------------------------------- |
| bg_color                  | String | *Optional*. Background color in the `#RRGGBB` format. Also available as the CSS variable `var(--tg-theme-bg-color)`. |
| text_color                | String | *Optional*. Main text color in the `#RRGGBB` format. Also available as the CSS variable `var(--tg-theme-text-color)`. |
| hint_color                | String | *Optional*. Hint text color in the `#RRGGBB` format. Also available as the CSS variable `var(--tg-theme-hint-color)`. |
| link_color                | String | *Optional*. Link color in the `#RRGGBB` format. Also available as the CSS variable `var(--tg-theme-link-color)`. |
| button_color              | String | *Optional*. Button color in the `#RRGGBB` format. Also available as the CSS variable `var(--tg-theme-button-color)`. |
| button_text_color         | String | *Optional*. Button text color in the `#RRGGBB` format. Also available as the CSS variable `var(--tg-theme-button-text-color)`. |
| secondary_bg_color        | String | *Optional*. Bot API 6.1+ Secondary background color in the `#RRGGBB` format. Also available as the CSS variable `var(--tg-theme-secondary-bg-color)`. |
| header_bg_color           | String | *Optional*. Bot API 7.0+ Header background color in the `#RRGGBB` format. Also available as the CSS variable `var(--tg-theme-header-bg-color)`. |
| bottom_bar_bg_color       | String | *Optional*. Bot API 7.10+ Bottom background color in the `#RRGGBB` format. Also available as the CSS variable `var(--tg-theme-bottom-bar-bg-color)`. |
| accent_text_color         | String | *Optional*. Bot API 7.0+ Accent text color in the `#RRGGBB` format. Also available as the CSS variable `var(--tg-theme-accent-text-color)`. |
| section_bg_color          | String | *Optional*. Bot API 7.0+ Background color for the section in the `#RRGGBB` format. It is recommended to use this in conjunction with *secondary_bg_color*. Also available as the CSS variable `var(--tg-theme-section-bg-color)`. |
| section_header_text_color | String | *Optional*. Bot API 7.0+ Header text color for the section in the `#RRGGBB` format. Also available as the CSS variable `var(--tg-theme-section-header-text-color)`. |
| section_separator_color   | String | *Optional*. Bot API 7.6+ Section separator color in the `#RRGGBB` format. Also available as the CSS variable `var(--tg-theme-section-separator-color)`. |
| subtitle_text_color       | String | *Optional*. Bot API 7.0+ Subtitle text color in the `#RRGGBB` format. Also available as the CSS variable `var(--tg-theme-subtitle-text-color)`. |
| destructive_text_color    | String | *Optional*. Bot API 7.0+ Text color for destructive actions in the `#RRGGBB` format. Also available as the CSS variable `var(--tg-theme-destructive-text-color)`. |



由于部分测试场景可能不存在telegram提供的基础css变量，所以要为其配置一套默认亮色及暗色主题，使用css媒体查询获取主题。
## 8.3 Telegram文档
Telegram的MiniApp文档参考 https://core.telegram.org/bots/webapps

---

## **9. UI 参考与验收**

* 参考需求截图：暗色背景、圆形亮色头像（首字母）、卡片阴影、搜索框带左侧放大镜图标。
* 验收需提供至少一张页面截图（搜索结果状态），与提供的参考视觉一致度高。
* 样式需适配桌面与移动端（响应式，搜索框和卡片在窄屏下保持可读）。

---

## **10. 交付物（Deliverables）**

* `prd/03-http-frontend-svelte.md`（本文件）及后续 TODO 清单。
* `http/frontend` 目录下的 SvelteKit 项目、生成的 API 客户端与构建脚本。
* 示例截图资源、README/运行说明、基础测试与 lint 配置。

