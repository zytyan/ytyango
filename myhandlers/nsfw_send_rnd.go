package myhandlers

import (
	"context"
	g "main/globalcfg"
	"regexp"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
)

var reRank = regexp.MustCompile(`(\d)(.*\b(\d))?`)

func SendRandRacy(bot *gotgbot.Bot, ctx *ext.Context) error {
	submatch := reRank.FindStringSubmatch(ctx.Message.Text)
	start, end := 2, 4
	if submatch != nil {
		start = defaultAtoi(submatch[1], 2)
		end = defaultAtoi(submatch[3], start)
	}
	end = max(start, end)
	start = min(start, end)

	start = max(start, 0)
	start = min(start, 6)

	end = max(end, 0)
	end = min(end, 6)

	end += 1
	photo, err := g.Q.GetPicByUserRateRange(context.Background(), start, end)
	if err != nil {
		_, _ = ctx.EffectiveMessage.Reply(bot, "没有涩图~", nil)
		return err
	}
	replyMarkup := BuildNsfwRateButton(photo.FileUid, "")
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
	if !chatCfg(msg.Chat.Id).RespNsfwMsg {
		return false
	}
	return reRacyPattern.MatchString(msg.Text)
}
