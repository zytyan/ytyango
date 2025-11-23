package myhandlers

import (
	"context"
	"fmt"
	g "main/globalcfg"
	"main/globalcfg/h"
	"regexp"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
)

var reRank = regexp.MustCompile(`(\d)(\b\d)?`)

func SendRandRacy(bot *gotgbot.Bot, ctx *ext.Context) error {
	submatch := reRank.FindStringSubmatch(ctx.Message.Text)
	start, end := 2, 2
	if submatch != nil {
		start = defaultAtoi(submatch[1], 2)
		end = defaultAtoi(submatch[2], start)
	}
	photo, err := g.Q.GetPicByUserRateRange(context.Background(), start, end)
	if err != nil {
		_, _ = ctx.EffectiveMessage.Reply(bot, "没有涩图~", nil)
		return err
	}
	cb := func(s int) string {
		return fmt.Sprintf("nsfw:%d:%s", s, photo.FileUid)
	}
	replyMarkup := h.NewInlineKeyboardButtonBuilder().
		Callback("不色！", cb(0)).
		Callback("有点涩", cb(2)).
		Row().
		Callback("好色哦", cb(4)).
		Callback("色爆了", cb(6)).
		Build()
	_, err = bot.SendPhoto(ctx.EffectiveChat.Id, gotgbot.InputFileByID(photo.FileID), &gotgbot.SendPhotoOpts{
		ReplyParameters: MakeReplyToMsgID(ctx.EffectiveMessage.MessageId),
		ReplyMarkup:     replyMarkup,
	})
	return err
}

var reRacyPattern = regexp.MustCompile(`来[张点个]([涩色瑟]|(se))图|再来一张|來[張點個]([澀色瑟]|(se))圖`)

func RequireNsfw(msg *gotgbot.Message) bool {
	if len(msg.Text) == 0 {
		return false
	}
	if !h.ChatRespNsfwMsg(msg.Chat.Id) {
		return false
	}
	return reRacyPattern.MatchString(msg.Text)
}
