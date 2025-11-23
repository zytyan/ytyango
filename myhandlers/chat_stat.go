package myhandlers

import (
	g "main/globalcfg"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
)

func StatMessage(bot *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	chat := ctx.EffectiveChat
	user := ctx.EffectiveUser
	if msg == nil || chat == nil || user == nil {
		return nil
	}
	chatStat := g.Q.ChatStatToday(chat.Id)
	chatStat.IncMessage(user.Id, int64(len(msg.Text)), msg.Date)
	if msg.Photo != nil {
		chatStat.IncPhotoCount()
	}
	if msg.Video != nil {
		chatStat.IncVideoCount()
	}
	if msg.Sticker != nil {
		chatStat.IncStickerCount()
	}
	if msg.ForwardOrigin != nil {
		chatStat.IncForwardCount()
	}
	return nil
}
