package myhandlers

import (
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
)

func OcrMessage(bot *gotgbot.Bot, ctx *ext.Context) error {
	if ctx.Message == nil {
		return NoImage
	}
	var photo *gotgbot.PhotoSize
	if ctx.Message.Photo != nil {
		photo = &ctx.Message.Photo[len(ctx.Message.Photo)-1]
	} else if ctx.Message.ReplyToMessage != nil && ctx.Message.ReplyToMessage.Photo != nil {
		photo = &ctx.Message.ReplyToMessage.Photo[len(ctx.Message.ReplyToMessage.Photo)-1]
	} else {
		return NoImage
	}
	content, err := ocrMsg(bot, photo)
	if err != nil {
		return err
	}
	_, err = ctx.EffectiveMessage.Reply(bot, content, nil)
	return err
}
