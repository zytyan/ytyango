package genai_hldr

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	g "main/globalcfg"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
	"google.golang.org/genai"
)

type searchArgs struct {
	Query    string
	Username string
	Limit    int
}

type searchResult struct {
	MsgID    int64  `json:"msg_id"`
	ChatID   int64  `json:"chat_id"`
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	MsgType  string `json:"msg_type"`
	SentTime string `json:"sent_time"`
	Text     string `json:"text"`
}

func (h *Handler) tools() []*genai.Tool {
	return []*genai.Tool{
		{GoogleSearch: &genai.GoogleSearch{}},
		{FunctionDeclarations: []*genai.FunctionDeclaration{h.searchFunctionDeclaration()}},
	}
}

func (h *Handler) searchFunctionDeclaration() *genai.FunctionDeclaration {
	max := float64(h.cfg.Search.MaxResults)
	min := float64(1)
	return &genai.FunctionDeclaration{
		Name:        searchToolName,
		Description: "在当前聊天的记录中按关键词或用户名检索最近消息，返回包含 user_id 与用户名的匹配列表",
		Parameters: &genai.Schema{
			Type: genai.TypeObject,
			Properties: map[string]*genai.Schema{
				"query": {
					Type:        genai.TypeString,
					Description: "查询关键词，支持文本或用户名片段",
				},
				"username": {
					Type:        genai.TypeString,
					Description: "可选的用户名过滤",
				},
				"limit": {
					Type:        genai.TypeInteger,
					Description: fmt.Sprintf("返回的最大结果数，默认不超过 %d", h.cfg.Search.MaxResults),
					Minimum:     &min,
					Maximum:     &max,
				},
			},
			Required: []string{"query"},
		},
		Response: &genai.Schema{
			Type: genai.TypeObject,
			Properties: map[string]*genai.Schema{
				"query": {
					Type:        genai.TypeString,
					Description: "本次查询关键词",
				},
				"limit": {
					Type:        genai.TypeInteger,
					Description: "本次查询使用的返回上限",
				},
				"matches": {
					Type: genai.TypeArray,
					Items: &genai.Schema{
						Type: genai.TypeObject,
						Properties: map[string]*genai.Schema{
							"msg_id":   {Type: genai.TypeInteger},
							"chat_id":  {Type: genai.TypeInteger},
							"user_id":  {Type: genai.TypeInteger},
							"username": {Type: genai.TypeString},
							"msg_type": {Type: genai.TypeString},
							"sent_time": {
								Type:        genai.TypeString,
								Description: "消息时间，RFC3339 格式",
							},
							"text": {
								Type:        genai.TypeString,
								Description: "消息文本或引用片段（截断后）",
							},
						},
					},
				},
				"error": {Type: genai.TypeString},
			},
		},
	}
}

func (h *Handler) handleFunctionCalls(ctx context.Context, session *Session, calls []*genai.FunctionCall) ([]*genai.Content, error) {
	responses := make([]*genai.Content, 0, len(calls))
	var firstErr error
	for _, call := range calls {
		switch call.Name {
		case searchToolName:
			part, note, err := h.handleSearchCall(ctx, session, call)
			if err != nil && firstErr == nil {
				firstErr = err
			}
			if part != nil {
				responses = append(responses, genai.NewContentFromParts([]*genai.Part{part}, genai.RoleUser))
			}
			if note != "" {
				session.appendModelText(note, "search_result")
			}
		default:
			h.log.Warnw("unknown function call", "name", call.Name, "session_id", session.ID, "chat_id", session.ChatID)
		}
	}
	return responses, firstErr
}

func (h *Handler) handleSearchCall(ctx context.Context, session *Session, call *genai.FunctionCall) (*genai.Part, string, error) {
	query := strings.TrimSpace(toString(call.Args["query"]))
	username := strings.TrimSpace(toString(call.Args["username"]))
	limit := toInt(call.Args["limit"])
	if limit <= 0 || limit > h.cfg.Search.MaxResults {
		limit = h.cfg.Search.MaxResults
	}
	args := searchArgs{
		Query:    query,
		Username: username,
		Limit:    limit,
	}
	matches, err := h.searchFn(ctx, session, args)
	resp := map[string]any{
		"query":   query,
		"limit":   limit,
		"matches": matches,
	}
	if err != nil {
		resp["error"] = err.Error()
	}
	part := genai.NewPartFromFunctionResponse(call.Name, resp)
	if part.FunctionResponse != nil {
		part.FunctionResponse.ID = call.ID
	}
	note := buildSearchNote(query, matches)
	return part, note, err
}

func (h *Handler) searchMessages(ctx context.Context, session *Session, args searchArgs) ([]searchResult, error) {
	keyword := strings.TrimSpace(args.Query)
	if args.Username != "" && !strings.Contains(strings.ToLower(keyword), strings.ToLower(args.Username)) {
		keyword = strings.TrimSpace(keyword + " " + args.Username)
	}
	limit := args.Limit
	if limit <= 0 || limit > h.cfg.Search.MaxResults {
		limit = h.cfg.Search.MaxResults
	}
	rows, err := g.Q.SearchGeminiContents(ctx, session.ChatID, keyword, keyword, int64(limit))
	if err != nil {
		return nil, err
	}
	results := make([]searchResult, 0, len(rows))
	for _, row := range rows {
		text := strings.TrimSpace(row.Text.String)
		if text == "" && row.QuotePart.Valid {
			text = strings.TrimSpace(row.QuotePart.String)
		}
		text = limitString(text, h.cfg.Search.MaxSnippet)
		userID, username := h.lookupUser(ctx, row.ChatID, row.MsgID, row.Username)
		results = append(results, searchResult{
			MsgID:    row.MsgID,
			ChatID:   row.ChatID,
			UserID:   userID,
			Username: username,
			MsgType:  row.MsgType,
			SentTime: row.SentTime.Time.Format(time.RFC3339),
			Text:     text,
		})
	}
	return results, nil
}

func (h *Handler) lookupUser(ctx context.Context, chatID, msgID int64, fallback string) (int64, string) {
	username := fallback
	var userID int64
	if g.Msgs != nil {
		saved, err := g.Msgs.GetSavedMessageById(ctx, chatID, msgID)
		if err == nil && saved.FromUserID.Valid {
			userID = saved.FromUserID.Int64
			if u, uErr := g.Q.GetUserById(ctx, userID); uErr == nil && u != nil {
				username = u.Name()
			} else if uErr != nil && !errors.Is(uErr, sql.ErrNoRows) {
				h.logD.Warn("lookup user", zap.Error(uErr))
			}
		} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
			h.logD.Warn("lookup saved message", zap.Error(err))
		}
	}
	if username == "" && userID != 0 {
		username = fmt.Sprintf("user_%d", userID)
	}
	return userID, username
}

func buildSearchNote(query string, matches []searchResult) string {
	builder := strings.Builder{}
	if query == "" {
		builder.WriteString("> 搜索结果：")
	} else {
		builder.WriteString("> 搜索结果(query=" + query + ")：")
	}
	if len(matches) == 0 {
		builder.WriteString("未找到匹配")
		return builder.String()
	}
	for _, m := range matches {
		builder.WriteString("\n- [")
		builder.WriteString(strconv.FormatInt(m.MsgID, 10))
		builder.WriteString("] ")
		builder.WriteString(m.Username)
		if m.UserID != 0 {
			builder.WriteString(" (")
			builder.WriteString(strconv.FormatInt(m.UserID, 10))
			builder.WriteString(")")
		}
		if m.Text != "" {
			builder.WriteString(": ")
			builder.WriteString(m.Text)
		}
	}
	return builder.String()
}

func toString(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case fmt.Stringer:
		return val.String()
	default:
		return fmt.Sprint(val)
	}
}

func toInt(v any) int {
	switch val := v.(type) {
	case int:
		return val
	case int64:
		return int(val)
	case float64:
		return int(val)
	case json.Number:
		i, _ := val.Int64()
		return int(i)
	default:
		return 0
	}
}
