# HTTP Frontend Search Loading TODO

- [x] 需求确认
  - [x] 核对 PRD 技术细节：状态字段、分页逻辑、IntersectionObserver 触发、加载/空态文案。

- [x] 交互与状态
  - [x] 在 `+page.svelte` 新增 `isLoadingInitial`/`isLoadingMore`/`hasMore`/`error` 等状态，搜索时重置分页与结果。
  - [x] 搜索请求添加等待动画（初次加载骨架或 spinner），不遮挡搜索框。

- [x] 无限滚动
  - [x] 在结果列表底部放置 sentinel，使用 `IntersectionObserver` 触发 `loadMore`，保证 `hasMore && !isLoadingMore` 时才请求。
  - [x] 将新页结果 append 至 `hits`，依据返回条数或 total 更新 `hasMore`。
  - [x] `hits.length == 0` 或 `!hasMore` 时在底部显示“没有更多了”，避免重复触发。

- [x] 错误处理与重试
  - [x] 底部展示加载失败提示与重试入口；重试继续当前分页。

- [x] 样式与主题
  - [x] 加载动画、提示文案使用 Telegram 主题变量，兼容移动端与桌面端。

- [ ] 验收与测试
  - [ ] 手动验证：初次搜索加载动画、滚动触底自动加载、空结果/末尾显示“没有更多了”、错误提示可重试。
  - [ ] 更新 TODO 完成度并提交。
