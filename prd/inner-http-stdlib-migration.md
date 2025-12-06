# **Product Requirements Document**

## **产品名称：Inner HTTP 服务标准库迁移**

## **版本：v1.1**

## **撰写日期：2025-12-07**

---

## **1. 背景（Background）**

当前 bot 的 `inner_http.go` 基于 Gin 运行，职责仅涵盖内部回调与日志调节：

* 运行在 `127.0.0.1:4019`，仅服务内部调用。
* 提供火星统计、Dio 统计、日志级别调整等少量路由。
* Gin 带来额外依赖体积与启动开销，错误恢复与日志格式也与其他标准库服务不一致。

目标是迁移为 Go 标准库 `net/http`，保留现有功能与接口契约，减少依赖并统一服务栈。

---

## **2. 目标（Objectives）**

* 移除对 Gin 的依赖，完全使用 `net/http` 提供同等功能。
* 保持现有路由与行为兼容（参数名、HTTP 状态码、返回格式）。
* 提供统一的访问日志与 panic 恢复逻辑，继续复用现有 zap logger。

---

## **3. 非目标（Out of Scope）**

* 不新增新路由或对外暴露端口。
* 不调整业务统计逻辑（`ChatStatToday` 相关的计数规则保持不变）。
* 不改动其他使用 Gin 的组件（仅限 `inner_http.go`）。

---

## **4. 用户角色（User Personas）**

### **Bot 维护者 / DevOps**

* 希望依赖更少、易于调试和部署。
* 需要快速调整日志级别、查看路由。

### **内部服务调用者（同进程或本机）**

* 通过 HTTP 调用统计接口，要求请求格式与返回码保持稳定。

---

## **5. 用户故事（User Stories）**

1. 作为维护者，我希望无需额外框架依赖即可启动内置 HTTP 服务，减少镜像体积与 CVE 面。
2. 作为运维，我希望仍能通过 `PUT /loggers/:name/:level` 快速调整日志级别，并收到明确的错误提示。
3. 作为内部调用方，我希望迁移后 `/mars-counter`、`/dio-ban` 的 JSON 请求格式与 400/200 行为不变。

---

## **6. 功能需求（Functional Requirements）**

### **6.1 路由与行为兼容**

| ID   | 描述                                                                 | 优先级 |
| ---- | -------------------------------------------------------------------- | --- |
| FR-1 | `POST /mars-counter` 接受 `{"group_id": int64,"mars_count": int64}`；校验失败返回 400，无 body；成功返回 200。 | 高 |
| FR-2 | `POST /dio-ban` 接受 `{"user_id": int64,"group_id": int64,"action": int}`；解析失败返回 400；成功返回 200 并触发对应计数。 | 高 |
| FR-3 | `GET /loggers` 返回纯文本列表，展示现有 logger 名称与级别，`Content-Type: text/plain`。 | 中 |
| FR-4 | `PUT /loggers/:name/:level` 按路径参数调整 logger 级别；若 logger 不存在返回文本提示并列出可用项。 | 高 |
| FR-5 | `GET /`（或 Any） 返回可用路由列表；保持现有文案格式。 | 中 |

### **6.2 日志与错误处理**

| ID   | 描述                                                                | 优先级 |
| ---- | ----------------------------------------------------------------- | --- |
| FR-6 | 全部请求输出访问日志，包含方法、路径、状态码、耗时，继续使用现有 zap logger。 | 高 |
| FR-7 | 捕获 handler 运行时 panic，返回 500 并记录错误堆栈，避免进程崩溃。     | 高 |

---

### **6.3 暴露Golang内置pprof**
| ID   | 描述                                                                | 优先级 |
|---|-----------------------------------|---|
|FR-8|基于 `net/http/pprof` 向接口提供prof工具| 高|

## **7. 非功能需求（Non-functional Requirements）**

### **可靠性**

* 服务异常时可恢复，不导致进程退出。
* JSON 解析使用有限大小的 `http.MaxBytesReader`，防止过大请求拖垮进程。

### **安全**

* 绑定默认为 `127.0.0.1:4019`，但可以使用环境变量 `BOT_INNER_HTTP`配置，若`BOT_INNER_HTTP`为`OFF`，则不监听。
* 日志不泄漏敏感字段（仅记录必要元信息）。

### **可维护性 / 可测试性**

* 新实现具备单元或集成测试覆盖核心路由（解析、状态码、logger 不存在等）。
* 代码遵循 `gofmt`，依赖更新后执行 `go mod tidy`。

---

## **8. 技术方案（Tech Design Summary）**

* 使用 `net/http` + `http.ServeMux` 或自定义路由器实现路由分发。
* 通过 `zap.Logger` 构建中间件：记录访问日志、捕获 panic、度量耗时。
* 使用 `json.Decoder` 解析请求体，启用 `DisallowUnknownFields`（如需保持宽松模式，则按现有行为实现）。
* 保留 `formatLoggers` 与 logger 查找逻辑，仅替换 Gin 取参、响应写入方式。
* 将 `Any("/")` 行为转换为 `ServeMux` 默认路由或专门 handler。

---

## **9. 数据结构（Data Models）**

### **MarsCounterRequest**

```go
type MarsInfo struct {
    GroupID   int64 `json:"group_id"`
    MarsCount int64 `json:"mars_count"`
}
```

### **DioBanRequest**

```go
type DioBanUser struct {
    UserId  int64 `json:"user_id"`
    GroupId int64 `json:"group_id"`
    Action  int   `json:"action"`
}
```

---

## **10. 里程碑（Milestones）**

| 时间 | 目标 |
| --- | --- |
| Day 1 | 完成 PRD 评审与确认，设计 handler 迁移方案。 |
| Day 2 | 实现 `net/http` 版本路由、日志与 panic 恢复；通过基础自测。 |
| Day 3 | 增补测试用例、`go test ./...` 验证，准备合并。 |

---

## **11. 风险（Risks）与对策（Mitigations）**

| 风险 | 影响 | 对策 |
| --- | --- | --- |
| 路由行为与 Gin 存在细微差异（状态码、Header） | 可能破坏现有调用 | 明确对齐现有行为，编写回归测试覆盖状态码与头信息。 |
| 日志格式变化影响下游采集 | 运维工具解析失败 | 在 PR 中展示新日志示例，必要时保持字段一致或兼容模式。 |
| JSON 解析宽松度差异 | 旧调用方可能发送多余字段 | 先维持宽松解析策略，并在文档中提示未来收敛。 |

---

## **12. 验收标准（Acceptance Criteria）**

* 所有既有路由返回与 Gin 版本一致的状态码与主体。
* panic 被捕获且记录，服务持续运行。
* `go test ./...` 通过；若外部依赖导致失败需在说明中记录。
* 当前仍有其他模块依赖gin，所以目前无需在 `go.mod` 中移除gin依赖

---
