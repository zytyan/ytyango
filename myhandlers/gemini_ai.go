package myhandlers

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	g "main/globalcfg"
	"main/globalcfg/h"
	"main/globalcfg/q"
	"strings"
	"sync"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
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
	geminiModel               = "gemini-2.5-flash-preview-09-2025"
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
	label += "-end-label-\n"
	out.Parts = append(out.Parts, &genai.Part{
		Text: label,
	})
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
		data, err = h.GetFileBytes(bot, msg.Photo[len(msg.Photo)-1].FileId)
		if err != nil {
			return err
		}
		content.MsgType = "photo"
		content.Blob = data
		content.MimeType.Valid = true
		content.MimeType.String = "image/jpeg"
	} else if msg.Sticker != nil {
		data, err = h.GetFileBytes(bot, msg.Sticker.FileId)
		if err != nil {
			return err
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
	Q, tx, err := g.NewTx()
	if err != nil {
		return err
	}
	for i := range s.TmpContents {
		err = s.TmpContents[i].Save(ctx, Q)
		if err != nil {
			return err
		}
	}
	s.Contents = append(s.Contents, s.TmpContents...)
	s.TmpContents = nil
	s.UpdateTime = time.Now()
	return tx.Commit()
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
		if time.Now().Sub(sess.UpdateTime) < geminiInterval {
			return sess
		}
		delete(geminiSessions.sidToSess, sess.ID)
	}
	delete(geminiSessions.chatIdToSess, msg.Chat.Id)
	var err error
	session.GeminiSession, err = g.Q.CreateNewGeminiSession(ctx, msg.Chat.Id, getChatName(&msg.Chat), msg.Chat.Type)
	err = session.loadContentFromDatabase(ctx)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	geminiSessions.sidToSess[session.ID] = session
	geminiSessions.chatIdToSess[msg.Chat.Id] = session
	return session
}

func GeminiReply(bot *gotgbot.Bot, ctx *ext.Context) error {
	client, err := getGenAiClient()
	if err != nil {
		return err
	}
	genCtx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()
	session := GeminiGetSession(genCtx, ctx.EffectiveMessage)
	if session == nil {
		return nil
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	sysInst := fmt.Sprintf(`now:%s
这里是一个Telegram聊天 type:%s,name:%s
你是一个Telegram机器人，name: %s username: %s
你会看到很多消息，每个消息头部都有一个元数据，以 '-start-label-'开头， '-end-label-' 结尾，元数据中标记了消息的ID(id)，发送时间(time)，发送者的用户名（name)以及消息类型(type)
消息类型有 text, photo, sticker三种，对应文本消息、图片消息及表情消息。
若用户明确回复了某条消息，则有回复的消息的ID(reply)字段。
这些元数据由代码自动生成，不要在模型的输出中加入该数据。
请使用中文回复消息。
不要使用markdown语法。`,
		time.Now().Format("2006-01-02 15:04:05 -07:00"),
		session.ChatType,
		session.ChatName,
		bot.FirstName+bot.LastName,
		bot.Username)
	config := &genai.GenerateContentConfig{
		SystemInstruction: genai.NewContentFromText(sysInst, genai.RoleModel),
		Tools: []*genai.Tool{
			{GoogleSearch: &genai.GoogleSearch{}},
			{URLContext: &genai.URLContext{}},
		},
	}
	err = session.AddTgMessage(bot, ctx.EffectiveMessage.ReplyToMessage)
	err = session.AddTgMessage(bot, ctx.EffectiveMessage)
	if err != nil {
		return err
	}
	_, err = ctx.EffectiveMessage.SetReaction(bot, &gotgbot.SetMessageReactionOpts{
		Reaction: []gotgbot.ReactionType{gotgbot.ReactionTypeEmoji{Emoji: "👀"}},
		IsBig:    false,
	})
	if err != nil {
		log.Warnf("set reaction emoji to message %s(%d) failed ", getChatName(&ctx.EffectiveMessage.Chat), ctx.EffectiveMessage.MessageId)
	}
	res, err := client.Models.GenerateContent(genCtx, geminiModel, session.ToGenaiContents(), config)
	if err != nil {
		_, _ = ctx.EffectiveMessage.SetReaction(bot, &gotgbot.SetMessageReactionOpts{
			Reaction: []gotgbot.ReactionType{gotgbot.ReactionTypeEmoji{Emoji: "😭"}},
			IsBig:    false,
		})
		_, _ = ctx.EffectiveMessage.Reply(bot, fmt.Sprintf("error:%s", err), nil)
		return err
	}
	respMsg, err := ctx.EffectiveMessage.Reply(bot, res.Text(), nil)
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
	user, err := g.Q.GetUserByTg(context.Background(), ctx.EffectiveUser)
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
