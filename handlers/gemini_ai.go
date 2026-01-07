package handlers

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	g "main/globalcfg"
	"main/globalcfg/h"
	"main/globalcfg/q"
	"main/helpers/mdnormalizer"
	"regexp"
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
	geminiSessionContentLimit = 100
	geminiModel               = "gemini-3-flash-preview"
	geminiInterval            = time.Minute * 3
)

type GeminiSession struct {
	q.GeminiSession
	mu          sync.Mutex
	Contents    []q.GeminiContent
	TmpContents []q.GeminiContent
	UpdateTime  time.Time
}

var geminiSessions struct {
	mu sync.RWMutex
	// session id -> session ，这是一个缓存
	sidToSess    map[int64]*GeminiSession
	chatIdToSess map[int64]*GeminiSession
}

func init() {
	geminiSessions.sidToSess = make(map[int64]*GeminiSession)
	geminiSessions.chatIdToSess = make(map[int64]*GeminiSession)
}
func databaseContentToGenaiPart(content *q.GeminiContent) (out *genai.Content) {
	out = &genai.Content{}
	label := fmt.Sprintf(`-start-label-
id:%d
time:%s
name:%s
type:%s
`, content.MsgID, content.SentTime.Format("2006-01-02 15:04:05"), content.Username, content.MsgType)
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
	content := q.GeminiContent{
		SessionID: s.ID,
		ChatID:    msg.Chat.Id,
		MsgID:     msg.MessageId,
		Role:      role,
		SentTime:  q.UnixTime{Time: time.Unix(msg.Date, 0)},
		Username:  msg.GetSender().Name(),
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
		data, err = h.DownloadToMemoryCached(bot, msg.Sticker.FileId)
		if err != nil {
			return err
		}
		if len(data) == 0 || data[0] == '{' {
			return errors.New("不支持的动画类型")
		}
		content.Blob = data
		content.MsgType = "sticker"
		content.MimeType.Valid = true
		if msg.Sticker.IsVideo {
			content.MimeType.String = "video/webm"
		} else {
			content.MimeType.String = "image/webp"
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

func GeminiGetSession(ctx context.Context, msg *gotgbot.Message) *GeminiSession {
	geminiSessions.mu.Lock()
	defer geminiSessions.mu.Unlock()
	session := &GeminiSession{}
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
		geminiSessions.chatIdToSess[msg.Chat.Id] = session
		return session
	}
create:
	sess, ok := geminiSessions.chatIdToSess[msg.Chat.Id]
	if ok {
		if time.Since(sess.UpdateTime) < geminiInterval {
			return sess
		}
		delete(geminiSessions.sidToSess, sess.ID)
	}
	delete(geminiSessions.chatIdToSess, msg.Chat.Id)
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
	geminiSessions.chatIdToSess[msg.Chat.Id] = session
	return session
}

var reLabelHeader = regexp.MustCompile(`(?s)^-start-label-\n.*-end-label-\n`)

func GeminiReply(bot *gotgbot.Bot, ctx *ext.Context) error {
	client, err := getGenAiClient()
	if !slices.Contains([]int64{-1001471592463, -1001282155019, -1001126241898, -1001170816274}, ctx.EffectiveChat.Id) {
		return nil
	}
	if err != nil {
		return err
	}
	genCtx, cancel := context.WithTimeout(context.Background(), time.Minute*5)
	defer cancel()
	session := GeminiGetSession(genCtx, ctx.EffectiveMessage)
	if session == nil {
		return nil
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	sysPrompt, err := g.Q.GetGeminiSystemPrompt(genCtx, ctx.EffectiveChat.Id)
	if errors.Is(err, sql.ErrNoRows) {
		sysPrompt = fmt.Sprintf(`现在是:%s
这里是一个Telegram聊天 type:%s,name:%s
你是一个Telegram机器人，name: %s username: %s
你会看到很多消息，每个消息头部都有一个元数据，以 '-start-label-'开头， '-end-label-' 结尾，元数据中标记了消息的ID(id)，发送时间(time)，发送者的用户名（name)以及消息类型(type)
消息类型有 text, photo, sticker三种，对应文本消息、图片消息及表情消息。
若用户明确回复了某条消息，则有回复的消息的ID(reply)字段。
若用户特地引用了被回复的消息中的某段文字，则会有引用(quote)字段。
这些元数据由代码自动生成，不要在模型的输出中加入该数据。`,
			time.Now().Format("2006-01-02 15:04:05 -07:00"),
			session.ChatType,
			session.ChatName,
			bot.FirstName+bot.LastName,
			bot.Username)
	} else if err != nil {
		return err
	}

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
	_, err = ctx.EffectiveMessage.SetReaction(bot, &gotgbot.SetMessageReactionOpts{
		Reaction: []gotgbot.ReactionType{gotgbot.ReactionTypeEmoji{Emoji: "👀"}},
		IsBig:    false,
	})
	if err != nil {
		log.Warnf("set reaction emoji to message %s(%d) failed ", getChatName(&ctx.EffectiveMessage.Chat), ctx.EffectiveMessage.MessageId)
	}
	defer func() {
		session.DiscardTmpUpdates()
	}()
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
	res, err := client.Models.GenerateContent(genCtx, geminiModel, session.ToGenaiContents(), config)
	tickerCancel()
	ticker.Stop()
	if err != nil {
		_, _ = ctx.EffectiveMessage.SetReaction(bot, &gotgbot.SetMessageReactionOpts{
			Reaction: []gotgbot.ReactionType{gotgbot.ReactionTypeEmoji{Emoji: "😭"}},
		})
		_, _ = ctx.EffectiveMessage.Reply(bot, fmt.Sprintf("error:%s", err), nil)
		return err
	}
	text := res.Text()
	text = reLabelHeader.ReplaceAllString(text, "")
	if text == "" {
		text = "模型没有返回任何信息"
		if res.PromptFeedback != nil {
			text += "，原因: " + string(res.PromptFeedback.BlockReason) + res.PromptFeedback.BlockReasonMessage
		}
		_, _ = ctx.EffectiveMessage.SetReaction(bot, &gotgbot.SetMessageReactionOpts{
			Reaction: []gotgbot.ReactionType{gotgbot.ReactionTypeEmoji{Emoji: "😭"}},
		})
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

func UpdateGeminiSysPrompt(bot *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	text := msg.GetText()
	prompt := h.TrimCmd(text)
	if prompt == "" {
		if msg.ReplyToMessage == nil || msg.ReplyToMessage.GetText() == "" {
			rawPrompt, err := g.Q.GetGeminiSystemPrompt(context.Background(), msg.Chat.Id)
			if err != nil {
				rawPrompt = "当前没有提示词"
			} else {
				rawPrompt = "当前提示词为" + rawPrompt
			}
			_, err = msg.Reply(bot, "没有找到任何System prompt，请使用 /sysprompt 提示词或使用该命令回复其他消息设置提示词。\n"+
				"使用 /resetsysprompt 恢复原始提示词\n"+rawPrompt, nil)
			return err
		}
	}
	err := g.Q.CreateOrUpdateGeminiSystemPrompt(context.Background(), msg.Chat.Id, prompt)
	if err != nil {
		_, err = msg.Reply(bot, "设置系统提示词错误: "+err.Error(), nil)
		return err
	}
	_, err = msg.Reply(bot, "成功设置系统提示词:\n"+prompt, nil)
	return err
}
func ResetGeminiSysPrompt(bot *gotgbot.Bot, ctx *ext.Context) error {
	err := g.Q.ResetGeminiSystemPrompt(context.Background(), ctx.EffectiveChat.Id)
	_, err = ctx.EffectiveMessage.Reply(bot, err.Error(), nil)
	return err
}
