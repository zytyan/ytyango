# **Product Requirements Document — HTTP Backend OpenAPI/Ogen Migration**

## **产品名称：HTTP Backend & Schema 重构**

## **版本：v1.0**

## **撰写日期：2025-12-07**


## **1. 背景（Background）**

当前 bot 的 HTTP 入口位于 `bothttp` 目录并基于 Gin，路由和业务混杂，后续计划加入 Svelte 前端（同处 `http` 目录）。需要迁移到 OpenAPI 驱动的代码生成（ogen），统一后端与未来前端共享的接口描述，并在迁移期精简功能，仅保留核心能力（搜索、用户信息、用户头像）。

---

## **2. 目标（Objectives）**

* 将 HTTP 后端迁移至 `http/backend`，使用 OpenAPI + ogen 生成服务器代码。
* OpenAPI schema 置于 `http` 目录下供后端与未来前端共享。
* 精简接口：仅保留搜索、用户信息（支持批量，最多 50）、用户头像三类能力。
* 头像验证从 header/query 迁移到路径参数形态：`.../{userId}?tgauth=xxx`。
* 搜索接口仅接受 JSON 请求体，不再支持 multipart 或 GET。
* 为核心路径补充自动化测试与必要的假数据，确保可回归。

---

## **3. 非目标（Out of Scope）**

* 不实现前端（Svelte）具体功能，仅为其预留目录结构(`http` base目录，该需求不涉及任何前端目录，不要新建`frontend`)与共享 schema。
* 不迁移或重写非核心业务功能（汇率、视频下载、跑团等）；相关路由移除。
* 不支持旧的 Gin 路由层；不保留 multipart/GET 搜索接口兼容层。

---

## **4. 用户角色（User Personas）**

### **后端开发者 / 维护者**

* 需要通过 OpenAPI 规范快速生成类型安全的服务端代码。
* 需要简化路由与处理器，降低维护复杂度。

### **Bot 集成方 / 调用方**

* 希望稳定获取搜索、用户信息、头像三类接口。
* 需要明确的认证位置（头像路径参数含 tgauth）。

### **QA / 验收人员**

* 需要可直接运行的测试与假数据，验证接口契约与边界（批量 50 上限、认证缺失等）。

---

## **5. 用户故事（User Stories）**

1. **作为后端维护者，我希望在 `http/backend` 中通过 ogen 自动生成的 server skeleton 接入业务逻辑，减少手写样板。**
2. **作为调用方，我希望搜索接口只需提交 JSON 请求体即可完成查询，不必关心 multipart/GET 兼容。**
3. **作为调用方，我希望一次批量获取多名用户信息，但数量上限得到明确校验（最多 50）。**
4. **作为调用方，我希望在获取头像时将 Telegram 验证信息放在路径参数上（`/{userId}?tgauth=...`），避免额外 header。**
5. **作为 QA，我希望存在自动化测试覆盖核心路径，并附带假数据以快速验证迁移后的行为。**

---

## **6. 功能需求（Functional Requirements）**

### **6.1 目录与生成流程**

| ID   | 描述                                                          | 优先级 |
| ---- | ----------------------------------------------------------- | --- |
| FR-1 | 新后端代码位于 `http/backend`，移除旧 `bothttp` 依赖。              | 高  |
| FR-2 | OpenAPI schema 放置于 `http` 根目录（如 `http/openapi.yaml`），由 ogen 生成 server 代码。 | 高  |
| FR-3 | 生成代码与手写实现分层，便于后续前端共享 schema。                            | 高  |

### **6.2 搜索接口**

| ID   | 描述                                                                  | 优先级 |
| ---- | ------------------------------------------------------------------- | --- |
| FR-4 | 接口仅接受 `application/json` 请求体，移除 multipart 与 GET 变种。                    | 高  |
| FR-5 | 请求体包含必要搜索字段（关键词、可选分页参数）；若格式错误返回 400。                         | 高  |
| FR-6 | 返回结构化 JSON 结果；必要时提供假数据以便测试。                                   | 中  |

### **6.3 用户信息接口**

| ID   | 描述                                                                 | 优先级 |
| ---- | ------------------------------------------------------------------ | --- |
| FR-7 | 支持批量查询用户信息，请求体/参数允许传入用户 ID 列表。                               | 高  |
| FR-8 | 批量数量上限 50；超过上限返回 400 并提示。                                        | 高  |
| FR-9 | 返回用户基本信息字段（如 id、name、username）。                       | 中  |

### **6.4 用户头像接口**

| ID    | 描述                                                                           | 优先级 |
| ----- | ---------------------------------------------------------------------------- | --- |
| FR-10 | 路由形如 `/users/{userId}/avatar?tgauth=...`，验证信息通过路径段和查询参数传入。           | 高  |
| FR-11 | 无 tgauth 或校验失败返回 401/403。                                                | 高  |
| FR-12 | 返回头像二进制。失败时返回404，由前端自动生成占位头像。                                      | 中  |

### **6.5 其他**

| ID    | 描述                                            | 优先级 |
| ----- | --------------------------------------------- | --- |
| FR-13 | 移除未启用功能对应的路由与处理器，保持最小可用集。              | 高  |
| FR-14 | 提供基础健康检查/版本信息（可选），便于部署验证。                | 低  |

---

## **7. 非功能需求（Non-functional Requirements）**

* **性能**：核心接口 p95 < 100ms（在假数据环境）；批量校验在 O(n) 内完成。
* **可维护性**：OpenAPI 规范与实现保持同步，提供脚本或 Make 目标生成 ogen 代码。
* **测试**：`go test ./...` 通过；新增接口需有单元/集成测试覆盖主要分支与边界条件（批量上限、认证缺失）。
* **可读性**：生成代码与手写业务逻辑分隔，目录结构清晰（如 `http/backend/ogen` vs `http/backend/handlers`）。

---

## **8. 技术方案（Tech Design Summary）**

* 使用 OpenAPI v3 schema（放置在 `http/openapi.yaml`）；通过 ogen 生成 server stub 与类型。
* 将运行入口与路由迁移至 `http/backend`；删除 Gin 依赖，使用 net/http + ogen 生成路由。
* 现有业务逻辑中仅保留搜索、用户信息、头像相关处理；其余 handler 移除或注释掉路由注册。
* 提供本地假数据（如 `http/backend/testdata/*.json`）和/或内存存根，便于测试搜索与用户信息路径。
* 搜索接口仅接受 JSON 解析；头像验证逻辑改为从路径/查询参数读取 tgauth。
* 提供 Make/脚本入口（如 `make gen-http` 或 `go generate`）运行 ogen（工具已安装）。

---

## **9. 数据结构（Data Models / API 契约示例）**

### 9.1 Telegram 验证方法
```go

type WebInitUser struct {
	Id              int    `json:"id"`
	FirstName       string `json:"first_name"`
	LastName        string `json:"last_name"`
	Username        string `json:"username"`
	LanguageCode    string `json:"language_code"`
	IsPremium       bool   `json:"is_premium"`
	AllowsWriteToPm bool   `json:"allows_write_to_pm"`
}

type AuthInfo struct {
	QueryId  string      `json:"query_id"`
	User     WebInitUser `json:"user"`
	AuthDate time.Time   `json:"auth_date"`
	Hash     string      `json:"hash"`
}

func checkTelegramAuth(str string, verifyKey []byte) (res AuthInfo, err error) {
	split := strings.Split(str, "&")
	const hashPrefix = "hash"
	recvHash := ""
	data := make([]string, 0, len(split))
	for _, v := range split {
		key, value, _ := strings.Cut(v, "=")
		if key == hashPrefix {
			recvHash = value
			continue
		}
		key, err1 := url.QueryUnescape(key)
		value, err2 := url.QueryUnescape(value)
		if err1 != nil || err2 != nil {
			err = fmt.Errorf("url unescape err %v %v", err1, err2)
			return
		}
		data = append(data, key+"="+value)
	}
	if recvHash == "" {
		err = fmt.Errorf("no hash")
		return
	}

	slices.Sort(data)
	initData := []byte(strings.Join(data, "\n"))
	mac := hmac.New(sha256.New, verifyKey)
	mac.Write(initData)
	calcHash := hex.EncodeToString(mac.Sum(nil))
	if recvHash != calcHash {
		err = fmt.Errorf("wrong recvHash calc=%s recv=%s", calcHash, recvHash)
		return
	}
	for _, v := range data {
		key, value, _ := strings.Cut(v, "=")
		switch key {
		case "auth_date":
			parseInt, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return AuthInfo{}, err
			}
			res.AuthDate = time.Unix(parseInt, 0)
		case "hash":
			res.Hash = value
		case "query_id":
			res.QueryId = value
		case "user":
			var user WebInitUser
			err = jsoniter.Unmarshal([]byte(value), &user)
			if err != nil {
				return
			}
			res.User = user
		}
	}
	return
}
```
验签密钥取自 `sha256.Sum256(botToken)`，调用端需传入未经重排的原始 querystring，避免再造假数据。
### 9.2 数据结构
#### 9.2.1 查找数据结构
```go
type SearchQuery struct {
	Query string    `json:"q"`
	InsID JsonInt64 `json:"ins_id"`
	Page  int       `json:"page"`
	Limit int       `json:"limit,omitempty"`
}
```
其中，JsonInt64为字符串表示的数字，可支持int64类型
#### 9.2.2 查找功能实现
查找功能已经在老版本中实现，且无关路由类型，复用该代码。
```go
type SearchResult struct {
	Hits []myhandlers.MeiliMsg `json:"hits"`

	Query              string `json:"query"`
	ProcessingTimeMs   int    `json:"processingTimeMs"`
	Limit              int    `json:"limit"`
	Offset             int    `json:"offset"`
	EstimatedTotalHits int    `json:"estimatedTotalHits"`
}

type MeiliQuery struct {
	Q      string   `json:"q"`
	Filter string   `json:"filter"`
	Limit  int      `json:"limit"`
	Offset int      `json:"offset"`
	Sort   []string `json:"sort"`
}

func meiliSearch(query SearchQuery) (SearchResult, error) {
	var result SearchResult
	searchUrl := g.GetConfig().MeiliConfig.GetSearchUrl()
	chat, err := g.Q.GetChatByWebId(context.Background(), int64(query.InsID))
	if chat == nil {
		return result, GroupNotFound
	}
	limit := query.GetLimit()
	meiliQuery := MeiliQuery{
		Q:      query.Query,
		Filter: "peer_id = " + strconv.FormatInt(chat.ID, 10),
		Limit:  limit,
		Offset: (query.Page - 1) * limit,
		Sort:   []string{"date:desc"},
	}
	data, err := jsoniter.Marshal(meiliQuery)
	if err != nil {
		return result, err
	}
	preparedPost, err := http.NewRequest(http.MethodPost, searchUrl, bytes.NewReader(data))
	if err != nil {
		return result, err
	}
	preparedPost.Header.Set("Authorization", "Bearer "+g.GetConfig().MeiliConfig.MasterKey)
	preparedPost.Header.Set("Content-Type", "application/json")
	post, err := http.DefaultClient.Do(preparedPost)
	if err != nil {
		return result, err
	}
	defer post.Body.Close()
	data, _ = io.ReadAll(post.Body)
	log.Infof("search result %s", data)
	err = jsoniter.Unmarshal(data, &result)
	if err != nil {
		return result, err
	}
	return result, nil
}
```

---
#### 9.2.3 用户数据结构体
用户结构体定义在 `globalcfg/q/models_gen.go:User`结构体中，但返回的数据中只需要 `User.Name()`，无需其他数据。
* **UserInfoRequest**：`POST /users/info <Headers> {"user_ids":[1,2,3...]}`
#### 9.2.4 用户头像
* **AvatarEndpoint**：`GET /users/{userId}/avatar?tgauth=token` → 二进制/redirect；401/403 on invalid tgauth。
由于用户头像可能需要bot动态下载，所以依然在此复用老代码。
```go
func webpConvert(in, out string) error {
	// if out path not exists, create it
	fp, err := os.Open(in)
	if err != nil {
		return err
	}
	defer fp.Close()
	img, _, err := image.Decode(fp)
	if err != nil {
		return err
	}
	outFp, err := os.Create(out)
	if err != nil {
		return err
	}
	defer outFp.Close()
	opt, err := encoder.NewLossyEncoderOptions(encoder.PresetDefault, 80)
	if err != nil {
		return err
	}
	err = webp.Encode(outFp, img, opt)
	if err != nil {
		return err
	}
	return nil
}
func getUserProfilePhotoWebp(userId int64) (string, error) {
	user := g.Q.GetUserById(context.Background(), userId)
	if user == nil {
		return "", UserNotFound
	}
	if !user.ProfilePhoto.Valid {
		return "", UserNoProfilePhoto
	}
	photoPath := fmt.Sprintf("data/profile_photo/p_%s.webp", user.ProfilePhoto.String)
	if fileExists(photoPath) {
		return photoPath, nil
	}
	path, err := user.DownloadProfilePhoto(myhandlers.GetMainBot())
	if err != nil {
		return "", err
	}
	err = webpConvert(path, photoPath)
	if err != nil {
		return "", err
	}
	return photoPath, nil
}

// 由于该代码在历史中被删除，所以重新放到这里
func (u *User) DownloadProfilePhoto(bot *gotgbot.Bot) (string, error) {
	if !u.ProfilePhoto.Valid || u.ProfilePhoto.String == "" {
		return "", errors.New("no profile photo")
	}
	file, err := bot.GetFile(u.ProfilePhoto.String, nil)
	if err != nil {
		return "", err
	}
	return file.FilePath, err
}
```
## **10. 里程碑（Milestones）**

| 时间      | 目标                                |
| -------- | --------------------------------- |
| Day 1    | 确立 OpenAPI schema & 目录结构；生成 ogen 基础代码 |
| Day 2    | 移植/实现搜索、用户信息、头像处理；移除旧路由           |
| Day 3    | 补充测试与假数据；`go test ./...` 通过               |

---

## **11. 风险（Risks）与对策（Mitigations）**

| 风险                           | 影响             | 对策                                  |
| ---------------------------- | -------------- | ----------------------------------- |
| 旧 Gin 代码与新生成代码差异大            | 迁移遗漏、回归风险   | 清理无关路由，基于 schema 重写入口；测试覆盖。          |
| OpenAPI 契约与实现偏离                 | 调用方集成失败      | 将 schema 作为单一事实来源，生成代码后再填充实现。        |
| 批量用户查询上限处理遗漏                | 可能耗尽资源或不一致 | 添加参数校验与测试覆盖边界（>50、空列表）。             |
| tgauth 迁移到路径后校验流程混乱          | 认证绕过            | 明确解析顺序与错误码，测试缺失/错误 token、缺少参数场景。 |

---

## **12. 验收标准（Acceptance Criteria）**

* `http/openapi.yaml` 存在并描述搜索、用户信息、头像接口；通过 ogen 生成服务器代码。
* 后端入口迁移至 `http/backend`，不再依赖 Gin；旧 `bothttp` 路由不再被调用。
* 搜索仅接受 JSON 请求体，其他格式拒绝；批量用户信息上限 50 并有错误返回。
* 头像接口使用 `/{userId}/avatar?tgauth=...` 进行验证并返回可用结果/占位。
* 自动化测试覆盖主要路径与异常情况，`go test ./...` 在本地通过。

---
