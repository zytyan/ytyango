package myhandlers

import (
	"context"
	"fmt"
	"main/globalcfg"
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

func IsGeminiReq(msg *gotgbot.Message) bool {
	return !strings.HasPrefix(msg.Text, "/") && strings.Contains(msg.Text, "@"+mainBot.Username)
}

func GeminiReply(bot *gotgbot.Bot, ctx *ext.Context) error {

	client, err := getGenAiClient()
	if err != nil {
		return err
	}
	genCtx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	sysInst := fmt.Sprintf(`time:%s
这里是一个Telegram聊天 type:%s,name:%s
请使用中文回复消息。
当前正处于原型测试阶段，不支持多轮对话。不要使用markdown语法。`, time.Now().Format("2006-01-02 15:04:05 -07:00"),
		ctx.EffectiveMessage.Chat.Type, getChatName(&ctx.EffectiveMessage.Chat))
	config := &genai.GenerateContentConfig{
		SystemInstruction: genai.NewContentFromText(sysInst, genai.RoleModel),
	}
	text := getText(ctx)
	//  "🤓", "👻", "👨‍💻", "👀",
	_, err = ctx.EffectiveMessage.SetReaction(bot, &gotgbot.SetMessageReactionOpts{
		Reaction: []gotgbot.ReactionType{gotgbot.ReactionTypeEmoji{Emoji: "👀"}},
		IsBig:    false,
	})
	if err != nil {
		log.Warnf("set reaction emoji to message %s(%d) failed ", getChatName(&ctx.EffectiveMessage.Chat), ctx.EffectiveMessage.MessageId)
	}
	res, err := client.Models.GenerateContent(genCtx, "gemini-2.5-flash-preview-09-2025", genai.Text(text), config)
	if err != nil {
		_, _ = ctx.EffectiveMessage.SetReaction(bot, &gotgbot.SetMessageReactionOpts{
			Reaction: []gotgbot.ReactionType{gotgbot.ReactionTypeEmoji{Emoji: "😭"}},
			IsBig:    false,
		})
		_, _ = ctx.EffectiveMessage.Reply(bot, fmt.Sprintf("error:%s", err), nil)
		return err
	}
	_, err = ctx.EffectiveMessage.Reply(bot, res.Text(), nil)
	return err
}
