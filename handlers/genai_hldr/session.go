package genai_hldr

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	g "main/globalcfg"
	"main/globalcfg/h"
	"main/globalcfg/q"
	"strings"
	"sync"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/dop251/goja"
	"google.golang.org/genai"
)

type Session struct {
	q.GeminiSession
	mu            sync.Mutex
	contents      []q.GeminiContent
	tmpContents   []q.GeminiContent
	updateTime    time.Time
	vm            *goja.Runtime
	replyBuf      strings.Builder
	virtualMsgID  int64
	botUsername   string
	botName       string
	maxCallStack  int
	maxReplyBytes int
}

func newSession(data q.GeminiSession, botName, botUsername string, maxCallStack, maxReplyBytes int) *Session {
	if strings.TrimSpace(botName) == "" {
		botName = botUsername
	}
	return &Session{
		GeminiSession: data,
		updateTime:    time.Now(),
		virtualMsgID:  -1,
		botName:       botName,
		botUsername:   botUsername,
		maxCallStack:  maxCallStack,
		maxReplyBytes: maxReplyBytes,
	}
}

func (s *Session) nextVirtualMsgID() int64 {
	id := s.virtualMsgID
	s.virtualMsgID--
	return id
}

func (s *Session) loadContentFromDatabase(ctx context.Context, limit int64) error {
	content, err := g.Q.GetAllMsgInSession(ctx, s.ID, limit)
	if err != nil {
		return err
	}
	s.contents = content
	return nil
}

func databaseContentToGenaiPart(content *q.GeminiContent) *genai.Content {
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
	out := &genai.Content{}
	out.Parts = append(out.Parts, &genai.Part{Text: label})
	if content.Text.Valid {
		out.Parts = append(out.Parts, &genai.Part{Text: content.Text.String})
	}
	if len(content.Blob) > 0 && content.MimeType.Valid {
		out.Parts = append(out.Parts, &genai.Part{InlineData: &genai.Blob{
			Data:     content.Blob,
			MIMEType: content.MimeType.String,
		}})
	}
	return out
}

func (s *Session) ToGenaiContents() []*genai.Content {
	contents := make([]*genai.Content, 0, len(s.contents)+len(s.tmpContents))
	for i := range s.contents {
		contents = append(contents, databaseContentToGenaiPart(&s.contents[i]))
	}
	for i := range s.tmpContents {
		contents = append(contents, databaseContentToGenaiPart(&s.tmpContents[i]))
	}
	return contents
}

func (s *Session) AddTelegramMessage(bot *gotgbot.Bot, msg *gotgbot.Message) error {
	if msg == nil {
		return nil
	}
	for i := range s.contents {
		if msg.MessageId == s.contents[i].MsgID {
			return nil
		}
	}
	for i := range s.tmpContents {
		if msg.MessageId == s.tmpContents[i].MsgID {
			return nil
		}
	}
	role := genai.RoleUser
	if msg.GetSender().Id() == bot.Id {
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
	if len(content.MsgType) == 0 {
		content.MsgType = "text"
	}
	var data []byte
	var err error
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
		content.Blob = data
		content.MsgType = "sticker"
		content.MimeType.Valid = true
		if msg.Sticker.IsVideo {
			content.MimeType.String = "video/webm"
		} else {
			content.MimeType.String = "image/webp"
		}
	}
	s.tmpContents = append(s.tmpContents, content)
	return nil
}

func (s *Session) PersistTmpUpdates(ctx context.Context) error {
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
	for i := range s.tmpContents {
		err = s.tmpContents[i].Save(ctx, newQ)
		if err != nil {
			return err
		}
	}
	s.contents = append(s.contents, s.tmpContents...)
	s.tmpContents = nil
	s.updateTime = time.Now()
	return tx.Commit()
}

func (s *Session) appendModelText(text string, msgType string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	content := q.GeminiContent{
		SessionID: s.ID,
		ChatID:    s.ChatID,
		MsgID:     s.nextVirtualMsgID(),
		Role:      genai.RoleModel,
		SentTime:  q.UnixTime{Time: time.Now()},
		Username:  s.botName,
		MsgType:   msgType,
	}
	content.Text = sql.NullString{String: text, Valid: true}
	s.tmpContents = append(s.tmpContents, content)
}

func (s *Session) ensureVM() error {
	if s.vm != nil {
		return nil
	}
	s.vm = goja.New()
	if s.maxCallStack > 0 {
		s.vm.SetMaxCallStackSize(s.maxCallStack)
	}
	err := s.vm.Set("reply", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) == 0 {
			panic(s.vm.NewGoError(errors.New("reply(payload) requires exactly 1 argument")))
		}
		payload := call.Arguments[0].Export()
		textBytes, err := json.Marshal(payload)
		if err != nil {
			panic(s.vm.NewGoError(err))
		}
		text := string(textBytes)
		if s.replyBuf.Len()+len(text) > s.maxReplyBytes {
			panic(s.vm.NewGoError(fmt.Errorf("reply output too large")))
		}
		s.replyBuf.WriteString(text)
		s.replyBuf.WriteString("\n")
		return goja.Undefined()
	})
	return err
}

type execTimeout struct{}

func (execTimeout) Error() string { return "js execution timeout" }

func (s *Session) RunJS(ctx context.Context, script string, limits ExecLimits) (string, error) {
	if limits.MaxScriptBytes > 0 && len(script) > limits.MaxScriptBytes {
		return "", fmt.Errorf("script too long: %d > %d bytes", len(script), limits.MaxScriptBytes)
	}
	if limits.MaxReplyBytes > 0 {
		s.maxReplyBytes = limits.MaxReplyBytes
	}
	if limits.MaxCallStack > 0 {
		s.maxCallStack = limits.MaxCallStack
	}
	if err := s.ensureVM(); err != nil {
		return "", err
	}
	s.vm.ClearInterrupt()
	s.replyBuf.Reset()
	var timer *time.Timer
	if limits.Timeout > 0 {
		timer = time.AfterFunc(limits.Timeout, func() {
			s.vm.Interrupt(execTimeout{})
		})
		defer timer.Stop()
	}
	cancelInterrupt := make(chan struct{})
	if ctx.Done() != nil {
		go func() {
			select {
			case <-ctx.Done():
				s.vm.Interrupt(ctx.Err())
			case <-cancelInterrupt:
				return
			}
		}()
	}
	defer close(cancelInterrupt)
	_, err := s.vm.RunString(script)
	if err != nil {
		if _, ok := err.(*goja.InterruptedError); ok {
			s.vm.ClearInterrupt()
		}
		return strings.TrimSpace(s.replyBuf.String()), err
	}
	return strings.TrimSpace(s.replyBuf.String()), nil
}
