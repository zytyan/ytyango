package handlers

import (
	"fmt"
	"main/globalcfg/h"
	"math/rand"
	"strconv"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
)

const callbackDataDelMe = "delme"

func getText(ctx *ext.Context) string {
	return getTextMsg(ctx.EffectiveMessage)
}

func getTextMsg(msg *gotgbot.Message) string {
	if msg == nil {
		return ""
	}
	if msg.Text != "" {
		return msg.Text
	}
	return msg.Caption
}

func parseIntDefault(str string, defaultValue int) int {
	if i, err := strconv.Atoi(str); err == nil {
		return i
	}
	return defaultValue
}

func Roll(bot *gotgbot.Bot, ctx *ext.Context) error {
	start, end := 1, 20
	args := ctx.Args()
	switch len(args) {
	case 1:
		break
	case 2:
		end = parseIntDefault(args[1], end)
	case 3:
		fallthrough
	default:
		start, end = parseIntDefault(args[1], start), parseIntDefault(args[2], end)
		if start > end {
			start, end = end, start
		}
	}
	rd := rand.Intn(end-start+1) + start
	reply := fmt.Sprintf("在%d-%d的roll点中，你掷出了%d\n本消息将在五分钟后删除", start, end, rd)
	_, err := ctx.Message.Reply(bot, reply,
		&gotgbot.SendMessageOpts{
			ReplyMarkup: h.NewInlineKeyboardButtonBuilder().Callback("删除该消息", callbackDataDelMe).Build(),
		})
	if err != nil {
		log.Warnf("reply failed: %s", err)
		return err
	}
	return nil
}

func DelMessage(bot *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	if msg == nil {
		return nil
	}
	_, err := msg.Delete(bot, nil)
	if err != nil {
		return err
	}
	rmsg := msg.ReplyToMessage
	if rmsg == nil {
		return nil
	}
	_, err = rmsg.Delete(bot, nil)
	return err
}

func IsDelMsgCallback(cb *gotgbot.CallbackQuery) bool {
	return cb.Data == callbackDataDelMe
}
