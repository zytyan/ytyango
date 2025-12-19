package handlers

import (
	"context"
	"fmt"
	g "main/globalcfg"
	"main/handlers/genai_hldr"
	"strings"
	"sync"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
)

var (
	geminiHandlerOnce sync.Once
	geminiHandler     *genai_hldr.Handler
)

func initGeminiHandler() {
	geminiHandlerOnce.Do(func() {
		h, err := genai_hldr.New(genai_hldr.Config{})
		if err != nil {
			panic(err)
		}
		geminiHandler = h
	})
}

func IsGeminiReq(msg *gotgbot.Message) bool {
	if msg == nil {
		return false
	}
	if strings.HasPrefix(msg.GetText(), "/") {
		return false
	}
	if mainBot != nil && strings.Contains(msg.GetText(), "@"+mainBot.Username) {
		return true
	}
	if msg.ReplyToMessage != nil {
		user := msg.ReplyToMessage.GetSender().User
		return user != nil && mainBot != nil && user.Id == mainBot.Id
	}
	return false
}

func GeminiReply(bot *gotgbot.Bot, ctx *ext.Context) error {
	initGeminiHandler()
	if geminiHandler == nil {
		return nil
	}
	return geminiHandler.Handle(bot, ctx)
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
