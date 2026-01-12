package handlers

import (
	"bytes"
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	g "main/globalcfg"
	"main/globalcfg/h"
	"main/globalcfg/q"
	"main/handlers/replacer"
	"main/helpers/mdnormalizer"
	"regexp"
	"slices"
	"strconv"
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
	geminiSessionContentLimit = 150
	geminiModel               = "gemini-3-flash-preview"
	geminiInterval            = time.Second * 30
	geminiMemoriesLimit       = 30
)

type GeminiSession struct {
	q.GeminiSession
	mu          sync.Mutex
	Contents    []q.GeminiContent
	TmpContents []q.GeminiContent
	UpdateTime  time.Time
	Memories    []q.GeminiMemory

	AllowCodeExecution bool
}
type geminiTopic struct {
	chatId  int64
	topicId int64
}

var geminiSessions struct {
	mu sync.RWMutex
	// session id -> session ，这是一个缓存
	sidToSess    map[int64]*GeminiSession
	chatIdToSess map[geminiTopic]*GeminiSession
}

func init() {
	geminiSessions.sidToSess = make(map[int64]*GeminiSession)
	geminiSessions.chatIdToSess = make(map[geminiTopic]*GeminiSession)
}
func databaseContentToGenaiPart(content *q.GeminiContent) (out *genai.Content) {
	out = &genai.Content{}
	label := fmt.Sprintf(`-start-label-
id:%d
time:%s
name:%s
type:%s
userid:%d
`, content.MsgID, content.SentTime.Format("2006-01-02 15:04:05"),
		content.Username,
		content.MsgType, content.UserID)
	if content.AtableUsername.Valid {
		label += fmt.Sprintf("username: %s", content.AtableUsername.String)
	}
	if content.ReplyToMsgID.Valid {
		label += fmt.Sprintf("reply:%d\n", content.ReplyToMsgID.Int64)
	}
	if content.QuotePart.Valid {
		label += fmt.Sprintf("quote:%s\n", content.QuotePart.String)
	}
	label += "-end-label-\n"
	out.Role = content.Role
	textPart := &genai.Part{
		Text: label,
	}
	out.Parts = append(out.Parts, textPart)
	if content.Text.Valid {
		out.Parts = append(out.Parts, &genai.Part{Text: content.Text.String})
	}
	if len(content.Blob) > 0 && content.MimeType.Valid {
		out.Parts = append(out.Parts, &genai.Part{InlineData: &genai.Blob{
			Data:     content.Blob,
			MIMEType: content.MimeType.String,
		}})
	}
	return
}

func (s *GeminiSession) ToGenaiContents() []*genai.Content {
	contents := make([]*genai.Content, 0, len(s.Contents)+1)
	for i := range s.Contents {
		contents = append(contents, databaseContentToGenaiPart(&s.Contents[i]))
	}
	for i := range s.TmpContents {
		contents = append(contents, databaseContentToGenaiPart(&s.TmpContents[i]))
	}
	return contents
}

func (s *GeminiSession) AddTgMessage(bot *gotgbot.Bot, msg *gotgbot.Message) (err error) {
	if msg == nil {
		return nil
	}
	for i := range s.Contents {
		if msg.MessageId == s.Contents[i].MsgID {
			return nil
		}
	}
	for i := range s.TmpContents {
		if msg.MessageId == s.TmpContents[i].MsgID {
			return nil
		}
	}
	role := genai.RoleUser
	if msg.GetSender().Id() == mainBot.Id {
		role = genai.RoleModel
	}
	username := msg.GetSender().Username()
	content := q.GeminiContent{
		SessionID:      s.ID,
		ChatID:         msg.Chat.Id,
		MsgID:          msg.MessageId,
		Role:           role,
		SentTime:       q.UnixTime{Time: time.Unix(msg.Date, 0)},
		Username:       msg.GetSender().Name(),
		AtableUsername: sql.NullString{String: username, Valid: username != ""},
		UserID:         msg.GetSender().Id(),
	}
	if msg.ReplyToMessage != nil {
		content.ReplyToMsgID.Valid = true
		content.ReplyToMsgID.Int64 = msg.ReplyToMessage.MessageId
		if msg.Quote != nil && msg.Quote.IsManual {
			content.QuotePart = sql.NullString{String: msg.Quote.Text, Valid: true}
		}
	}
	if msg.Text != "" {
		content.Text.Valid = true
		content.Text.String = msg.Text
		content.MsgType = "text"
	}
	if msg.Caption != "" {
		content.Text.Valid = true
		content.Text.String = msg.Caption
	}
	var data []byte
	if msg.Photo != nil {
		data, err = h.DownloadToMemoryCached(bot, msg.Photo[len(msg.Photo)-1].FileId)
		if err != nil {
			return err
		}
		content.MsgType = "photo"
		content.Blob = data
		content.MimeType.Valid = true
		content.MimeType.String = "image/jpeg"
	} else if msg.Sticker != nil {
		if msg.Sticker.IsAnimated {
			return errors.New("不支持的动画类型")
		}
		data, err = h.DownloadToMemoryCached(bot, msg.Sticker.FileId)
		if err != nil {
			return err
		}
		content.Blob = data
		content.MsgType = "sticker"
		content.MimeType.Valid = true
		if msg.Sticker.IsVideo {
			s.AllowCodeExecution = false
			content.MimeType.String = "video/webm"
		} else {
			content.MimeType.String = "image/webp"
		}
	} else if msg.Video != nil {
		if msg.Video.Duration <= 120 && msg.Video.FileSize <= 10*1024*1024 {
			s.AllowCodeExecution = false
			data, err = h.DownloadToMemoryCached(bot, msg.Video.FileId)
			if err != nil {
				return err
			}
			content.Blob = data
			content.MsgType = "video"
			content.MimeType.Valid = true
			content.MimeType.String = "video/mp4"
		}
	}
	s.TmpContents = append(s.TmpContents, content)
	return
}

func (s *GeminiSession) loadContentFromDatabase(ctx context.Context) error {
	content, err := g.Q.GetAllMsgInSession(ctx, s.ID, geminiSessionContentLimit)
	if err != nil {
		return err
	}
	s.AllowCodeExecution = true
	for _, c := range content {
		if c.MimeType.Valid && strings.Contains(c.MimeType.String, "video") {
			s.AllowCodeExecution = false
		}
	}
	s.Contents = content
	return nil
}

func (s *GeminiSession) PersistTmpUpdates(ctx context.Context) error {
	if len(s.TmpContents) == 0 {
		return nil
	}
	tx, err := g.RawMainDb().BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	newQ := g.Q.WithTx(tx)
	for i := range s.TmpContents {
		err = s.TmpContents[i].Save(ctx, newQ)
		if err != nil {
			return err
		}
	}
	s.Contents = append(s.Contents, s.TmpContents...)
	s.TmpContents = nil
	s.UpdateTime = time.Now()
	return tx.Commit()
}
func (s *GeminiSession) DiscardTmpUpdates() {
	s.TmpContents = nil
}

func IsGeminiReq(msg *gotgbot.Message) bool {
	text := msg.GetText()
	if strings.HasPrefix(text, "/") {
		return false
	}
	if strings.Contains(text, "@"+mainBot.Username) {
		return true
	}
	if msg.ReplyToMessage != nil {
		return msg.ReplyToMessage.GetSender().Id() == mainBot.Id
	}
	return false
}

func GeminiGetSession(ctx context.Context, msg *gotgbot.Message) *GeminiSession {
	geminiSessions.mu.Lock()
	defer geminiSessions.mu.Unlock()
	session := &GeminiSession{}
	topic := geminiTopic{
		chatId:  msg.Chat.Id,
		topicId: msg.MessageThreadId,
	}
	if msg.ReplyToMessage != nil {
		sessionId, err := g.Q.GetSessionIdByMessage(ctx, msg.Chat.Id, msg.ReplyToMessage.MessageId)
		if err == nil {
			if sess, ok := geminiSessions.sidToSess[sessionId]; ok {
				return sess
			}
		}
		session.GeminiSession, err = g.Q.GetSessionById(ctx, sessionId)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				goto create
			}
			return nil
		}
		err = session.loadContentFromDatabase(ctx)
		if err != nil {
			return nil
		}
		geminiSessions.sidToSess[sessionId] = session
		geminiSessions.chatIdToSess[topic] = session
		return session
	}
create:
	sess, ok := geminiSessions.chatIdToSess[topic]
	if ok {
		if time.Since(sess.UpdateTime) < geminiInterval {
			return sess
		}
		delete(geminiSessions.sidToSess, sess.ID)
	}
	delete(geminiSessions.chatIdToSess, topic)
	var err error
	session.GeminiSession, err = g.Q.CreateNewGeminiSession(ctx, msg.Chat.Id, getChatName(&msg.Chat), msg.Chat.Type)
	if err != nil {
		return nil
	}
	err = session.loadContentFromDatabase(ctx)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	geminiSessions.sidToSess[session.ID] = session
	geminiSessions.chatIdToSess[topic] = session
	return session
}

var reLabelHeader = regexp.MustCompile(`(?s)^-start-label-\n.*-end-label-\n`)

//go:embed gemini_sysprompt.txt
var gDefaultSysPrompt string
var geminiSysPromptReplacer = replacer.NewReplacer(gDefaultSysPrompt)
var sysPromptReplacerCache = make(map[geminiTopic]*replacer.Replacer)

var gMu sync.Mutex

func getSysPrompt(chatId, threadId int64) *replacer.Replacer {
	gMu.Lock()
	defer gMu.Unlock()
	topic := geminiTopic{
		chatId:  chatId,
		topicId: threadId,
	}
	if r, ok := sysPromptReplacerCache[topic]; ok {
		return r
	}
	tmpl, err := g.Q.GetGeminiSystemPrompt(context.Background(), chatId, threadId)
	if err == nil {
		r := replacer.NewReplacer(tmpl)
		sysPromptReplacerCache[topic] = &r
		return &r
	}
	sysPromptReplacerCache[topic] = &geminiSysPromptReplacer
	return &geminiSysPromptReplacer
}

func setReaction(bot *gotgbot.Bot, msg *gotgbot.Message, emoji string) {
	_, err := msg.SetReaction(bot, &gotgbot.SetMessageReactionOpts{
		Reaction: []gotgbot.ReactionType{gotgbot.ReactionTypeEmoji{Emoji: emoji}},
	})
	if err != nil {
		logD.Warn("set reaction", zap.Error(err))
	}
}

var reMemManager = regexp.MustCompile(`(?m)^/mem(add|edit|del).*?$`)

func GeminiReply(bot *gotgbot.Bot, ctx *ext.Context) error {
	if !slices.Contains([]int64{-1001471592463, -1001282155019, -1001126241898,
		-1001170816274, -1003612476571}, ctx.EffectiveChat.Id) {
		return nil
	}
	msg := ctx.EffectiveMessage
	topicId := int64(0)
	if msg.IsTopicMessage {
		topicId = msg.MessageThreadId
	}
	setReaction(bot, msg, "👀")
	client, err := getGenAiClient()
	if err != nil {
		return err
	}
	genCtx, cancel := context.WithTimeout(context.Background(), time.Minute*5)
	defer cancel()
	session := GeminiGetSession(genCtx, ctx.EffectiveMessage)
	if session == nil {
		return nil
	}
	if len(session.Memories) == 0 {
		memories, err := g.Q.ListGeminiMemory(genCtx, msg.Chat.Id, topicId, 30)
		if err != nil {
			return err
		}
		session.Memories = memories
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	sysPromptCtx := replacer.ReplaceCtx{
		Bot: bot,
		Msg: ctx.EffectiveMessage,
		Now: time.Now(),
	}
	for _, mem := range session.Memories {
		sysPromptCtx.Memories = append(sysPromptCtx.Memories, mem.Content)
	}
	sysPrompt := getSysPrompt(ctx.EffectiveChat.Id, ctx.EffectiveMessage.MessageThreadId).Replace(&sysPromptCtx)
	config := &genai.GenerateContentConfig{
		SystemInstruction: genai.NewContentFromText(sysPrompt, genai.RoleModel),
		Tools: []*genai.Tool{
			{GoogleSearch: &genai.GoogleSearch{}},
		},
	}
	if err := session.AddTgMessage(bot, ctx.EffectiveMessage.ReplyToMessage); err != nil {
		return err
	}
	if err := session.AddTgMessage(bot, ctx.EffectiveMessage); err != nil {
		return err
	}
	if session.AllowCodeExecution {
		config.Tools[0].CodeExecution = &genai.ToolCodeExecution{}
	}
	defer session.DiscardTmpUpdates()

	actionCancel := h.WithChatAction(bot, "typing", msg.Chat.Id, msg.MessageThreadId, msg.IsTopicMessage)
	defer actionCancel()
	res, err := client.Models.GenerateContent(genCtx, geminiModel, session.ToGenaiContents(), config)
	actionCancel()
	if err != nil {
		setReaction(bot, msg, "😭")
		_, _ = ctx.EffectiveMessage.Reply(bot, fmt.Sprintf("error:%s", err), nil)
		return err
	}
	_ = g.Q.IncrementSessionTokenCounters(
		genCtx,
		int64(res.UsageMetadata.PromptTokenCount),
		int64(res.UsageMetadata.CandidatesTokenCount),
		session.ID,
	)
	text := res.Text()
	text = reLabelHeader.ReplaceAllString(text, "")
	matches := reMemManager.FindAllString(text, -1)
	for _, match := range matches {
		if strings.HasPrefix(match, "/memadd ") {
			match = strings.TrimPrefix(match, "/memadd ")
			mem, err := g.Q.CreateGeminiMemory(genCtx, msg.Chat.Id, topicId, match)
			if err != nil {
				continue
			}
			session.Memories = append(session.Memories, mem)
		} else if strings.HasPrefix(match, "/memedit ") {
			fields := strings.SplitN(match, " ", 2)
			if len(fields) <= 2 {
				continue
			}
			id, err := strconv.ParseInt(fields[1], 10, 64)
			if err != nil || int(id) > len(session.Memories) || id <= 0 {
				continue
			}
			session.Memories[id-1].Content = fields[2]
			_ = g.Q.UpdateGeminiMemory(genCtx, session.Memories[id-1].ID, fields[2])
		} else if strings.HasPrefix(match, "/memdel ") {
			fields := strings.Fields(match)
			if len(fields) <= 1 {
				continue
			}
			for _, field := range fields[1:] {
				id, err := strconv.ParseInt(field, 10, 64)
				if err != nil || int(id) > len(session.Memories) {
					continue
				}
				session.Memories[id-1].Content = "<deleted>"
				_ = g.Q.DeleteGeminiMemory(genCtx, session.Memories[id-1].ID)
			}
		}
	}
	if text == "" {
		text = "模型没有返回任何信息"
		if res.PromptFeedback != nil {
			text += "，原因: " + string(res.PromptFeedback.BlockReason) + res.PromptFeedback.BlockReasonMessage
		}
		setReaction(bot, msg, "🤯")
		session.DiscardTmpUpdates()
	}
	normTxt, err := mdnormalizer.Normalize(text)
	var respMsg *gotgbot.Message
	if err != nil {
		respMsg, err = ctx.EffectiveMessage.Reply(bot, text, nil)
		logD.Warn("parse markdown failed", zap.Error(err))
	} else {
		respMsg, err = ctx.EffectiveMessage.Reply(bot, normTxt.Text, &gotgbot.SendMessageOpts{Entities: normTxt.Entities})
	}
	if err != nil {
		j, err2 := res.MarshalJSON()
		log.Warnf("genemi response: %s, error: %s", j, err2)
		return err
	}
	err = session.AddTgMessage(bot, respMsg)
	if err != nil {
		return err
	}
	return session.PersistTmpUpdates(genCtx)
}

func UpdateGeminiSysPrompt(bot *gotgbot.Bot, ctx *ext.Context) error {
	delete(sysPromptReplacerCache, geminiTopic{chatId: ctx.EffectiveChat.Id, topicId: ctx.EffectiveMessage.MessageThreadId})
	msg := ctx.EffectiveMessage
	text := msg.GetText()
	prompt := h.TrimCmd(text)
	if prompt == "" {
		if msg.ReplyToMessage == nil || msg.ReplyToMessage.GetText() == "" {
			_, err := msg.Reply(bot, `没有找到任何System prompt，请使用 /sysprompt 提示词或使用该命令回复其他消息设置提示词。
您需要使用 /get_sysprompt 获取当前系统提示词， /reset_sysprompt 恢复默认系统提示词。

你可以通过 %VAR% 使用变量，它会自动替换变量名，可使用的变量如下。
TIME: 当前时间，不包含日期
DATE: 当前日期，不含时间
DATETIME: 当前时间和日期
DATETIME_TZ: 包含时区的时间和日期
CHAT_NAME: 当前聊天的名称
BOT_NAME: Bot的名字
BOT_USERNAME: Bot的username
CHAT_TYPE: 聊天类型(group, private)

例：现在是%DATETIME%，当前聊天为%CHAT_NAME%，请根据需要解答群友的问题。
`, nil)
			return err
		}
	}
	err := g.Q.CreateOrUpdateGeminiSystemPrompt(context.Background(), msg.Chat.Id, msg.MessageThreadId, prompt)
	if err != nil {
		_, err = msg.Reply(bot, "设置系统提示词错误: "+err.Error(), nil)
		return err
	}
	_, err = msg.Reply(bot, "成功设置系统提示词:\n"+prompt, nil)
	return err
}
func ResetGeminiSysPrompt(bot *gotgbot.Bot, ctx *ext.Context) error {
	delete(sysPromptReplacerCache, geminiTopic{chatId: ctx.EffectiveChat.Id, topicId: ctx.EffectiveMessage.MessageThreadId})
	err := g.Q.ResetGeminiSystemPrompt(context.Background(), ctx.EffectiveChat.Id, ctx.EffectiveMessage.MessageThreadId)
	if err != nil {
		_, err = ctx.EffectiveMessage.Reply(bot, err.Error(), nil)
		return err
	}
	_, err = ctx.EffectiveMessage.Reply(bot, "已恢复默认提示词", nil)
	return err
}
func GetGeminiSysPrompt(bot *gotgbot.Bot, ctx *ext.Context) error {
	prompt, err := g.Q.GetGeminiSystemPrompt(context.Background(), ctx.EffectiveChat.Id, ctx.EffectiveMessage.MessageThreadId)
	if err != nil {
		_, err = ctx.EffectiveMessage.Reply(bot, gDefaultSysPrompt, nil)
		return err
	}
	_, err = ctx.EffectiveMessage.Reply(bot, prompt, nil)
	return err
}

func NewGeminiSession(bot *gotgbot.Bot, ctx *ext.Context) error {
	geminiSessions.mu.Lock()
	delete(geminiSessions.chatIdToSess, geminiTopic{
		chatId:  ctx.EffectiveMessage.GetChat().Id,
		topicId: ctx.EffectiveMessage.MessageThreadId,
	})
	geminiSessions.mu.Unlock()
	_, err := ctx.EffectiveMessage.Reply(bot, "已重新开始session，新建会话不会携带历史记录。", nil)
	return err
}

func GetGeminiSessionId(bot *gotgbot.Bot, ctx *ext.Context) error {
	geminiSessions.mu.Lock()
	sess := geminiSessions.chatIdToSess[geminiTopic{
		chatId:  ctx.EffectiveMessage.GetChat().Id,
		topicId: ctx.EffectiveMessage.MessageThreadId,
	}]
	geminiSessions.mu.Unlock()
	_, err := ctx.EffectiveMessage.Reply(bot, fmt.Sprintf("Session ID: %d", sess.ID), nil)
	return err
}

func GetMemories(bot *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	topicId := int64(0)
	if msg.IsTopicMessage {
		topicId = msg.MessageThreadId
	}
	memories, err := g.Q.ListGeminiMemory(context.Background(), msg.Chat.Id, topicId, geminiMemoriesLimit)
	if err != nil {
		_, _ = msg.Reply(bot, err.Error(), nil)
		return err
	}
	if len(memories) == 0 {
		_, err = msg.Reply(bot, "当前没有任何记忆", nil)
		return err
	}
	buf := bytes.NewBuffer(nil)
	for i, m := range memories {
		buf.WriteString(fmt.Sprintf("%d. %s\n", i+1, m.Content))
	}
	_, err = msg.Reply(bot, buf.String(), nil)
	return err
}
