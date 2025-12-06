package myhandlers

import (
	"context"
	"database/sql"
	"fmt"
	g "main/globalcfg"
	"main/globalcfg/q"
	"strings"
	"sync"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"google.golang.org/genai"
)

const geminiModel = "gemini-2.5-flash-preview-09-2025"

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

func geminiTools() []*genai.Tool {
	return []*genai.Tool{
		{URLContext: &genai.URLContext{}},
		{CodeExecution: &genai.ToolCodeExecution{}},
		{GoogleSearch: &genai.GoogleSearch{}},
	}
}

func IsGeminiReq(msg *gotgbot.Message) bool {
	if msg == nil {
		return false
	}
	text := getTextMsg(msg)
	if strings.HasPrefix(text, "/") {
		return false
	}
	return messageMentionsBot(msg) || isReplyToBotOrSelf(msg)
}

func GeminiReply(bot *gotgbot.Bot, ctx *ext.Context) error {

	client, err := getGenAiClient()
	if err != nil {
		return err
	}
	genCtx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	if ctx.EffectiveMessage == nil || ctx.EffectiveSender == nil {
		return nil
	}
	userID := ctx.EffectiveSender.Id()
	if userID == 0 && ctx.EffectiveUser != nil {
		userID = ctx.EffectiveUser.Id
	}
	mentioned := messageMentionsBot(ctx.EffectiveMessage)

	session, replySeq, err := resolveGeminiSession(context.Background(), ctx.EffectiveMessage, userID, mentioned)
	if err != nil {
		return err
	}
	if mentioned {
		rememberGeminiSession(ctx.EffectiveMessage.Chat.Id, session.ID)
	}

	cleanText := sanitizeGeminiText(getText(ctx))
	if cleanText == "" {
		cleanText = strings.TrimSpace(getText(ctx))
	}
	//  "🤓", "👻", "👨‍💻", "👀",
	_, err = ctx.EffectiveMessage.SetReaction(bot, &gotgbot.SetMessageReactionOpts{
		Reaction: []gotgbot.ReactionType{gotgbot.ReactionTypeEmoji{Emoji: "👀"}},
		IsBig:    false,
	})
	if err != nil {
		log.Warnf("set reaction emoji to message %s(%d) failed ", getChatName(&ctx.EffectiveMessage.Chat), ctx.EffectiveMessage.MessageId)
	}
	lastSeq, err := g.Q.GetGeminiLastSeq(context.Background(), session.ID)
	if err != nil {
		return err
	}
	userSeq := lastSeq + 1
	now := time.Now()
	_, err = g.Q.CreateGeminiMessage(context.Background(), q.CreateGeminiMessageParams{
		SessionID:   session.ID,
		ChatID:      ctx.EffectiveMessage.Chat.Id,
		TgMessageID: ctx.EffectiveMessage.MessageId,
		FromID:      userID,
		Role:        geminiRoleUser,
		Content:     cleanText,
		Seq:         userSeq,
		ReplyToSeq:  replySeq,
		CreatedAt:   q.UnixTime{Time: now},
	})
	if err != nil {
		return err
	}
	_ = g.Q.TouchGeminiSession(context.Background(), q.UnixTime{Time: now}, session.ID)

	history, err := g.Q.ListGeminiMessages(context.Background(), session.ID, geminiHistoryLimit)
	if err != nil {
		return err
	}
	sysInst := fmt.Sprintf(`time:%s
这里是一个Telegram聊天 type:%s,name:%s。
对话历史使用紧凑的ID格式：[u数字]代表用户消息，[b数字->数字]代表机器人消息以及它回复的消息ID。ID格式为程序自动编号，不要在输出中包含该格式。
保持中文回复，不要使用Markdown，直接给出答案。`, time.Now().Format("2006-01-02 15:04:05 -07:00"),
		ctx.EffectiveMessage.Chat.Type, getChatName(&ctx.EffectiveMessage.Chat))
	config := &genai.GenerateContentConfig{
		SystemInstruction: genai.NewContentFromText(sysInst, genai.RoleModel),
		Tools:             geminiTools(),
	}

	contents := compactGeminiHistory(history)
	res, err := callGeminiWithRetry(genCtx, client, contents, config)
	if err != nil {
		_, _ = ctx.EffectiveMessage.SetReaction(bot, &gotgbot.SetMessageReactionOpts{
			Reaction: []gotgbot.ReactionType{gotgbot.ReactionTypeEmoji{Emoji: "😭"}},
			IsBig:    false,
		})
		_, _ = ctx.EffectiveMessage.Reply(bot, fmt.Sprintf("error:%s", err), nil)
		return err
	}
	replyText := strings.TrimSpace(res.Text())
	botSeq := userSeq + 1
	sentMsg, sendErr := ctx.EffectiveMessage.Reply(bot, replyText, nil)
	if sendErr != nil {
		return sendErr
	}
	now = time.Now()
	_, err = g.Q.CreateGeminiMessage(context.Background(), q.CreateGeminiMessageParams{
		SessionID:   session.ID,
		ChatID:      ctx.EffectiveMessage.Chat.Id,
		TgMessageID: sentMsg.MessageId,
		FromID:      bot.Id,
		Role:        geminiRoleModel,
		Content:     replyText,
		Seq:         botSeq,
		ReplyToSeq:  sql.NullInt64{Int64: userSeq, Valid: true},
		CreatedAt:   q.UnixTime{Time: now},
	})
	if err != nil {
		log.Warnf("save gemini reply failed chat %d msg %d: %v", ctx.EffectiveMessage.Chat.Id, sentMsg.MessageId, err)
	}
	_ = g.Q.TouchGeminiSession(context.Background(), q.UnixTime{Time: now}, session.ID)
	return err
}

func callGeminiWithRetry(ctx context.Context, client *genai.Client, contents []*genai.Content, config *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
	var lastErr error
	backoff := geminiBackoffBase
	for i := 0; i < geminiMaxRetry; i++ {
		res, err := client.Models.GenerateContent(ctx, geminiModel, contents, config)
		if err == nil {
			return res, nil
		}
		lastErr = err
		if i == geminiMaxRetry-1 {
			break
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
		}
		backoff *= 2
	}
	return nil, lastErr
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
