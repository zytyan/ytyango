package myhandlers

import (
	"main/globalcfg/h"
	"regexp"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
)

func SendRandAdult(bot *gotgbot.Bot, ctx *ext.Context) error {
	photo := getRandomNsfwAdult()
	if photo == "" {
		_, err := ctx.EffectiveMessage.Reply(bot, "没有色图~", nil)
		return err
	}
	_, err := bot.SendPhoto(ctx.EffectiveChat.Id, gotgbot.InputFileByID(photo), &gotgbot.SendPhotoOpts{
		ReplyParameters: MakeReplyToMsgID(ctx.EffectiveMessage.MessageId),
		HasSpoiler:      true,
	})
	return err
}

func SendRandRacy(bot *gotgbot.Bot, ctx *ext.Context) error {
	photo := getRandomNsfwRacy()
	if photo == "" {
		_, err := ctx.EffectiveMessage.Reply(bot, "没有涩图~", nil)
		return err
	}
	_, err := bot.SendPhoto(ctx.EffectiveChat.Id, gotgbot.InputFileByID(photo), &gotgbot.SendPhotoOpts{
		ReplyParameters: MakeReplyToMsgID(ctx.EffectiveMessage.MessageId),
	})
	return err
}

var reRacyPattern = regexp.MustCompile(`来[张点个]([涩色瑟]|(se))图|再来一张|來[張點個]([澀色瑟]|(se))圖`)

func IsRequiredAdult(msg *gotgbot.Message) bool {
	if len(msg.Text) == 0 {
		return false
	}
	if !h.ChatRespNsfwMsg(msg.Chat.Id) {
		return false
	}
	return false
}

func IsRequiredRacy(msg *gotgbot.Message) bool {
	if len(msg.Text) == 0 {
		return false
	}
	if !h.ChatRespNsfwMsg(msg.Chat.Id) {
		return false
	}
	return reRacyPattern.MatchString(msg.Text)
}
