package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	g "main/globalcfg"
	"main/globalcfg/h"
	"main/globalcfg/q"
	"main/mdnormalizer"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"go.uber.org/zap"
	"google.golang.org/genai"
)

var getGenAiClient = sync.OnceValues(func() (*genai.Client, error) {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  g.GetConfig().GeminiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, err
	}
	return client, nil
})

const (
	geminiModel        = "gemini-3-flash-preview"
	geminiCacheTTL     = 6 * time.Hour
	geminiReplyTimeout = 120 * time.Second
)

type GeminiSession struct {
	q.GeminiSession
	mu      sync.Mutex
	chat    *genai.Chat
	nextSeq int64
}

var geminiSessions struct {
	mu     sync.RWMutex
	byChat map[int64]*GeminiSession
}

func init() {
	geminiSessions.byChat = make(map[int64]*GeminiSession)
}

func IsGeminiReq(msg *gotgbot.Message) bool {
	if strings.HasPrefix(msg.GetText(), "/") {
		return false
	}
	if strings.Contains(msg.GetText(), "@"+mainBot.Username) {
		return true
	}
	if msg.ReplyToMessage != nil {
		user := msg.ReplyToMessage.GetSender().User
		return user != nil && user.Id == mainBot.Id
	}
	return false
}

type pendingPart struct {
	Text                   sql.NullString
	Thought                bool
	ThoughtSignature       []byte
	InlineData             []byte
	InlineDataMime         sql.NullString
	FileURI                sql.NullString
	FileMime               sql.NullString
	FunctionCallName       sql.NullString
	FunctionCallArgs       sql.NullString
	FunctionResponseName   sql.NullString
	FunctionResponse       sql.NullString
	ExecutableCode         sql.NullString
	ExecutableCodeLanguage sql.NullString
	CodeExecutionOutcome   sql.NullString
	CodeExecutionOutput    sql.NullString
	VideoStartOffset       sql.NullString
	VideoEndOffset         sql.NullString
	VideoFps               sql.NullFloat64
	XUserExtra             sql.NullString
}

type pendingContent struct {
	content *genai.Content
	extra   sql.NullString
	parts   []pendingPart
}

func defaultGeminiTools() []*genai.Tool {
	return []*genai.Tool{
		{GoogleSearch: &genai.GoogleSearch{}, CodeExecution: &genai.ToolCodeExecution{}},
	}
}

func marshalTools(tools []*genai.Tool) sql.NullString {
	if len(tools) == 0 {
		return sql.NullString{}
	}
	b, err := json.Marshal(tools)
	if err != nil {
		return sql.NullString{}
	}
	return sql.NullString{String: string(b), Valid: true}
}

func resolveTools(session *GeminiSession) []*genai.Tool {
	if session.Tools.Valid && session.Tools.String != "" {
		var tools []*genai.Tool
		err := json.Unmarshal([]byte(session.Tools.String), &tools)
		if err == nil && len(tools) > 0 {
			return tools
		}
		logD.Warn("failed to parse gemini tools, fallback to default", zap.Error(err))
	}
	return defaultGeminiTools()
}

func getGeminiSession(ctx context.Context, msg *gotgbot.Message) (*GeminiSession, error) {
	geminiSessions.mu.RLock()
	if sess := geminiSessions.byChat[msg.Chat.Id]; sess != nil {
		geminiSessions.mu.RUnlock()
		return sess, nil
	}
	geminiSessions.mu.RUnlock()

	session, err := g.Q.GetGeminiSessionByChat(ctx, msg.Chat.Id)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		tools := defaultGeminiTools()
		session, err = g.Q.CreateGeminiSession(ctx, q.CreateGeminiSessionParams{
			ChatID:   msg.Chat.Id,
			ChatName: getChatName(&msg.Chat),
			ChatType: msg.Chat.Type,
			Tools:    marshalTools(tools),
		})
		if err != nil {
			return nil, err
		}
	}
	nextSeq, err := g.Q.NextGeminiSeq(ctx, session.ID)
	if err != nil {
		return nil, err
	}
	sess := &GeminiSession{
		GeminiSession: session,
		nextSeq:       nextSeq,
	}
	geminiSessions.mu.Lock()
	geminiSessions.byChat[msg.Chat.Id] = sess
	geminiSessions.mu.Unlock()
	return sess, nil
}

func buildSystemInstruction(bot *gotgbot.Bot, chat *gotgbot.Chat) *genai.Content {
	text := fmt.Sprintf(`现在是:%s
这里是一个Telegram聊天 type:%s,name:%s
你是一个Telegram机器人，name:%s username:%s
请使用中文回复消息，保持简洁，不要输出内部调试信息。`,
		time.Now().Format("2006-01-02 15:04:05 -07:00"),
		chat.Type,
		getChatName(chat),
		bot.FirstName+bot.LastName,
		bot.Username)
	return genai.NewContentFromText(text, genai.RoleModel)
}

func makePendingPart(part *genai.Part) (pendingPart, error) {
	p := pendingPart{
		Thought:          part.Thought,
		ThoughtSignature: part.ThoughtSignature,
	}
	extra := make(map[string]any)

	if part.Text != "" {
		p.Text = sql.NullString{String: part.Text, Valid: true}
	}
	if part.InlineData != nil {
		p.InlineData = part.InlineData.Data
		p.InlineDataMime = sql.NullString{String: part.InlineData.MIMEType, Valid: part.InlineData.MIMEType != ""}
	}
	if part.FileData != nil {
		p.FileURI = sql.NullString{String: part.FileData.FileURI, Valid: part.FileData.FileURI != ""}
		p.FileMime = sql.NullString{String: part.FileData.MIMEType, Valid: part.FileData.MIMEType != ""}
		if part.FileData.DisplayName != "" {
			extra["file_display_name"] = part.FileData.DisplayName
		}
	}
	if part.FunctionCall != nil {
		p.FunctionCallName = sql.NullString{String: part.FunctionCall.Name, Valid: part.FunctionCall.Name != ""}
		if len(part.FunctionCall.Args) > 0 {
			b, err := json.Marshal(part.FunctionCall.Args)
			if err != nil {
				return p, err
			}
			p.FunctionCallArgs = sql.NullString{String: string(b), Valid: true}
		}
	}
	if part.FunctionResponse != nil {
		p.FunctionResponseName = sql.NullString{String: part.FunctionResponse.Name, Valid: part.FunctionResponse.Name != ""}
		if len(part.FunctionResponse.Response) > 0 {
			b, err := json.Marshal(part.FunctionResponse.Response)
			if err != nil {
				return p, err
			}
			p.FunctionResponse = sql.NullString{String: string(b), Valid: true}
		}
	}
	if part.ExecutableCode != nil {
		p.ExecutableCode = sql.NullString{String: part.ExecutableCode.Code, Valid: part.ExecutableCode.Code != ""}
		p.ExecutableCodeLanguage = sql.NullString{String: string(part.ExecutableCode.Language), Valid: part.ExecutableCode.Language != ""}
	}
	if part.CodeExecutionResult != nil {
		p.CodeExecutionOutcome = sql.NullString{String: string(part.CodeExecutionResult.Outcome), Valid: part.CodeExecutionResult.Outcome != ""}
		p.CodeExecutionOutput = sql.NullString{String: part.CodeExecutionResult.Output, Valid: part.CodeExecutionResult.Output != ""}
	}
	if part.VideoMetadata != nil {
		if part.VideoMetadata.StartOffset != 0 {
			p.VideoStartOffset = sql.NullString{String: part.VideoMetadata.StartOffset.String(), Valid: true}
		}
		if part.VideoMetadata.EndOffset != 0 {
			p.VideoEndOffset = sql.NullString{String: part.VideoMetadata.EndOffset.String(), Valid: true}
		}
		if part.VideoMetadata.FPS != nil {
			p.VideoFps = sql.NullFloat64{Float64: *part.VideoMetadata.FPS, Valid: true}
		}
	}
	if part.MediaResolution != nil {
		extra["media_resolution"] = part.MediaResolution
	}
	if len(extra) > 0 {
		b, err := json.Marshal(extra)
		if err != nil {
			return p, err
		}
		p.XUserExtra = sql.NullString{String: string(b), Valid: true}
	}
	return p, nil
}

func pendingFromGenaiContent(content *genai.Content, extra map[string]any) (pendingContent, error) {
	if content == nil {
		return pendingContent{}, nil
	}
	pc := pendingContent{content: content}
	for _, part := range content.Parts {
		p, err := makePendingPart(part)
		if err != nil {
			return pendingContent{}, err
		}
		pc.parts = append(pc.parts, p)
	}
	if len(extra) > 0 {
		b, err := json.Marshal(extra)
		if err != nil {
			return pendingContent{}, err
		}
		pc.extra = sql.NullString{String: string(b), Valid: true}
	}
	return pc, nil
}

func pendingFromMessage(bot *gotgbot.Bot, msg *gotgbot.Message) (pendingContent, error) {
	if msg == nil {
		return pendingContent{}, errors.New("nil message")
	}
	content := &genai.Content{Role: genai.RoleUser}
	meta := map[string]any{
		"chat_id":  msg.Chat.Id,
		"msg_id":   msg.MessageId,
		"username": msg.GetSender().Name(),
		"date":     msg.Date,
	}
	if msg.ReplyToMessage != nil {
		meta["reply_to_msg_id"] = msg.ReplyToMessage.MessageId
		if msg.Quote != nil && msg.Quote.IsManual {
			meta["quote"] = msg.Quote.Text
		}
	}
	text := msg.Text
	if text == "" {
		text = msg.Caption
	}
	if text != "" {
		content.Parts = append(content.Parts, genai.NewPartFromText(text))
	}
	if msg.Photo != nil {
		data, err := h.DownloadToMemoryCached(bot, msg.Photo[len(msg.Photo)-1].FileId)
		if err != nil {
			return pendingContent{}, err
		}
		content.Parts = append(content.Parts, genai.NewPartFromBytes(data, "image/jpeg"))
		meta["msg_type"] = "photo"
	}
	if msg.Sticker != nil {
		data, err := h.DownloadToMemoryCached(bot, msg.Sticker.FileId)
		if err != nil {
			return pendingContent{}, err
		}
		mime := "image/webp"
		if msg.Sticker.IsVideo {
			mime = "video/webm"
		}
		content.Parts = append(content.Parts, genai.NewPartFromBytes(data, mime))
		meta["msg_type"] = "sticker"
	}
	pc, err := pendingFromGenaiContent(content, meta)
	if err != nil {
		return pendingContent{}, err
	}
	return pc, nil
}

func (s *GeminiSession) ensureChat(ctx context.Context, client *genai.Client, systemInstruction *genai.Content, tools []*genai.Tool, history []*genai.Content) (*genai.Chat, error) {
	if s.chat != nil {
		return s.chat, nil
	}
	cfg := &genai.GenerateContentConfig{
		SystemInstruction: systemInstruction,
		Tools:             tools,
	}
	if s.CacheName.Valid && s.CacheExpired.Int64 == 0 {
		cfg.CachedContent = s.CacheName.String
	}
	chat, err := client.Chats.Create(ctx, geminiModel, cfg, history)
	if err != nil {
		return nil, err
	}
	s.chat = chat
	return chat, nil
}

func (s *GeminiSession) markCacheExpired(ctx context.Context) {
	s.CacheExpired = sql.NullInt64{Int64: 1, Valid: true}
	_ = g.Q.UpdateGeminiSessionCache(ctx, q.UpdateGeminiSessionCacheParams{
		Tools:        s.Tools,
		CacheName:    s.CacheName,
		CacheTtl:     s.CacheTtl,
		CacheExpired: s.CacheExpired,
		ID:           s.ID,
	})
}

func (s *GeminiSession) refreshCache(ctx context.Context, client *genai.Client, tools []*genai.Tool, systemInstruction *genai.Content) {
	if s.chat == nil {
		return
	}
	cache, err := client.Caches.Create(ctx, geminiModel, &genai.CreateCachedContentConfig{
		TTL:               geminiCacheTTL,
		Contents:          s.chat.History(true),
		SystemInstruction: systemInstruction,
		Tools:             tools,
	})
	if err != nil {
		logD.Warn("failed to refresh gemini cache", zap.Error(err))
		return
	}
	s.CacheName = sql.NullString{String: cache.Name, Valid: cache.Name != ""}
	s.CacheTtl = sql.NullInt64{Int64: int64(geminiCacheTTL.Seconds()), Valid: true}
	s.CacheExpired = sql.NullInt64{Int64: 0, Valid: true}
	s.Tools = marshalTools(tools)
	if err := g.Q.UpdateGeminiSessionCache(ctx, q.UpdateGeminiSessionCacheParams{
		Tools:        s.Tools,
		CacheName:    s.CacheName,
		CacheTtl:     s.CacheTtl,
		CacheExpired: s.CacheExpired,
		ID:           s.ID,
	}); err != nil {
		logD.Warn("update gemini cache meta failed", zap.Error(err))
	}
}

func persistPending(ctx context.Context, qtx *q.Queries, sessionID int64, seq int64, pc pendingContent) error {
	if pc.content == nil {
		return nil
	}
	role := pc.content.Role
	if role == "" {
		role = genai.RoleUser
	}
	row, err := qtx.AddGeminiContentV2(ctx, sessionID, role, seq, pc.extra)
	if err != nil {
		return err
	}
	for idx, part := range pc.parts {
		if err := qtx.AddGeminiContentV2Part(ctx, q.AddGeminiContentV2PartParams{
			ContentID:              row.ID,
			PartIndex:              int64(idx),
			Text:                   part.Text,
			Thought:                part.Thought,
			ThoughtSignature:       part.ThoughtSignature,
			InlineData:             part.InlineData,
			InlineDataMime:         part.InlineDataMime,
			FileUri:                part.FileURI,
			FileMime:               part.FileMime,
			FunctionCallName:       part.FunctionCallName,
			FunctionCallArgs:       part.FunctionCallArgs,
			FunctionResponseName:   part.FunctionResponseName,
			FunctionResponse:       part.FunctionResponse,
			ExecutableCode:         part.ExecutableCode,
			ExecutableCodeLanguage: part.ExecutableCodeLanguage,
			CodeExecutionOutcome:   part.CodeExecutionOutcome,
			CodeExecutionOutput:    part.CodeExecutionOutput,
			VideoStartOffset:       part.VideoStartOffset,
			VideoEndOffset:         part.VideoEndOffset,
			VideoFps:               part.VideoFps,
			XUserExtra:             part.XUserExtra,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *GeminiSession) persistExchange(ctx context.Context, user pendingContent, model pendingContent) error {
	tx, err := g.RawMainDb().BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	qtx := g.Q.WithTx(tx)
	nextSeq, err := qtx.NextGeminiSeq(ctx, s.ID)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := persistPending(ctx, qtx, s.ID, nextSeq, user); err != nil {
		_ = tx.Rollback()
		return err
	}
	nextSeq++
	if model.content != nil {
		if err := persistPending(ctx, qtx, s.ID, nextSeq, model); err != nil {
			_ = tx.Rollback()
			return err
		}
		nextSeq++
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	s.nextSeq = nextSeq
	return nil
}

func GeminiReply(bot *gotgbot.Bot, ctx *ext.Context) error {
	client, err := getGenAiClient()
	if err != nil {
		return err
	}
	if !slices.Contains([]int64{-1001471592463, -1001282155019, -1001126241898, -1001170816274}, ctx.EffectiveChat.Id) {
		return nil
	}
	genCtx, cancel := context.WithTimeout(context.Background(), geminiReplyTimeout)
	defer cancel()
	session, err := getGeminiSession(genCtx, ctx.EffectiveMessage)
	if err != nil {
		return err
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	tools := resolveTools(session)
	sysInst := buildSystemInstruction(bot, &ctx.EffectiveMessage.Chat)
	chat, err := session.ensureChat(genCtx, client, sysInst, tools, nil)
	if err != nil {
		return err
	}
	userPending, err := pendingFromMessage(bot, ctx.EffectiveMessage)
	if err != nil {
		return err
	}
	if len(userPending.content.Parts) == 0 {
		return errors.New("empty content for gemini")
	}
	_, err = ctx.EffectiveMessage.SetReaction(bot, &gotgbot.SetMessageReactionOpts{
		Reaction: []gotgbot.ReactionType{gotgbot.ReactionTypeEmoji{Emoji: "👀"}},
		IsBig:    false,
	})
	if err != nil {
		log.Warnf("set reaction emoji to message %s(%d) failed ", getChatName(&ctx.EffectiveMessage.Chat), ctx.EffectiveMessage.MessageId)
	}
	ticker := time.NewTicker(time.Second * 4)
	defer ticker.Stop()
	tickerCtx, tickerCancel := context.WithCancel(context.Background())
	defer tickerCancel()
	go func() {
		_, _ = bot.SendChatAction(ctx.EffectiveChat.Id, "typing", nil)
		for {
			select {
			case <-ticker.C:
				_, _ = bot.SendChatAction(ctx.EffectiveChat.Id, "typing", nil)
			case <-tickerCtx.Done():
				return
			}
		}
	}()
	res, err := chat.Send(genCtx, userPending.content.Parts...)
	if err != nil && session.CacheName.Valid {
		history := chat.History(true)
		session.markCacheExpired(genCtx)
		session.chat = nil
		chat, err = session.ensureChat(genCtx, client, sysInst, tools, history)
		if err == nil {
			res, err = chat.Send(genCtx, userPending.content.Parts...)
		}
	}
	tickerCancel()
	ticker.Stop()
	if err != nil {
		_, _ = ctx.EffectiveMessage.SetReaction(bot, &gotgbot.SetMessageReactionOpts{
			Reaction: []gotgbot.ReactionType{gotgbot.ReactionTypeEmoji{Emoji: "😭"}},
		})
		_, _ = ctx.EffectiveMessage.Reply(bot, fmt.Sprintf("error:%s", err), nil)
		return err
	}
	var modelContent *genai.Content
	if len(res.Candidates) > 0 && res.Candidates[0].Content != nil {
		modelContent = res.Candidates[0].Content
	}
	respText := res.Text()
	if respText == "" {
		respText = "模型没有返回任何信息"
		if res.PromptFeedback != nil {
			respText += "，原因: " + string(res.PromptFeedback.BlockReason) + res.PromptFeedback.BlockReasonMessage
		}
		_, _ = ctx.EffectiveMessage.SetReaction(bot, &gotgbot.SetMessageReactionOpts{
			Reaction: []gotgbot.ReactionType{gotgbot.ReactionTypeEmoji{Emoji: "😭"}},
		})
	}
	normTxt, normErr := mdnormalizer.Normalize(respText)
	var respMsg *gotgbot.Message
	if normErr != nil {
		respMsg, err = ctx.EffectiveMessage.Reply(bot, respText, nil)
		logD.Warn("parse markdown failed", zap.Error(normErr))
	} else {
		respMsg, err = ctx.EffectiveMessage.Reply(bot, normTxt.Text, &gotgbot.SendMessageOpts{Entities: normTxt.Entities})
	}
	if err != nil {
		j, err2 := res.MarshalJSON()
		log.Warnf("genemi response: %s, error: %s", j, err2)
		return err
	}

	modelMeta := map[string]any{
		"response_id": res.ResponseID,
		"model":       res.ModelVersion,
		"chat_id":     respMsg.Chat.Id,
		"msg_id":      respMsg.MessageId,
	}
	if session.CacheName.Valid && session.CacheExpired.Int64 == 0 {
		modelMeta["cached_content"] = session.CacheName.String
	}
	if res.UsageMetadata != nil {
		modelMeta["prompt_token_count"] = res.UsageMetadata.PromptTokenCount
		modelMeta["response_token_count"] = res.UsageMetadata.CandidatesTokenCount
		modelMeta["cached_content_token_count"] = res.UsageMetadata.CachedContentTokenCount
	}
	modelPending, err := pendingFromGenaiContent(modelContent, modelMeta)
	if err != nil {
		return err
	}
	if err := session.persistExchange(genCtx, userPending, modelPending); err != nil {
		return err
	}
	go func() {
		cacheCtx, cancel := context.WithTimeout(context.Background(), time.Second*15)
		defer cancel()
		session.refreshCache(cacheCtx, client, tools, sysInst)
	}()
	return nil
}

func SetUserTimeZone(bot *gotgbot.Bot, ctx *ext.Context) error {
	fields := strings.Fields(ctx.EffectiveMessage.Text)
	const help = "用法: /settimezone +0800"
	if len(fields) < 2 {
		_, err := ctx.EffectiveMessage.Reply(bot, help, nil)
		return err
	}
	t, err := time.Parse("-0700", fields[1])
	if err != nil {
		_, err := ctx.EffectiveMessage.Reply(bot, help, nil)
		return err
	}
	_, zone := t.Zone()
	user, err := g.Q.GetOrCreateUserByTg(context.Background(), ctx.EffectiveUser)
	if err != nil {
		return err
	}
	err = g.Q.UpdateUserTimeZone(context.Background(), user, int64(zone))
	if err != nil {
		_, err = ctx.EffectiveMessage.Reply(bot, err.Error(), nil)
		return err
	}
	user.Timezone = int64(zone)
	_, err = ctx.EffectiveMessage.Reply(bot, fmt.Sprintf("设置成功 %d seconds", zone), nil)
	return err
}
