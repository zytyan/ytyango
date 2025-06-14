package myhandlers

import (
	"bytes"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/disintegration/imaging"
	"main/helpers/imgproc"
)

func IsSacabam(msg *gotgbot.Message) bool {
	return imgproc.MatchSacabambaspis(msg.Text)
}

func GenSacabam(bot *gotgbot.Bot, ctx *ext.Context) error {
	img := imgproc.GenSacaImage(ctx.EffectiveMessage.Text)
	buf := &bytes.Buffer{}
	err := imaging.Encode(buf, img, imaging.JPEG)
	if err != nil {
		return err
	}
	_, err = bot.SendPhoto(ctx.EffectiveChat.Id,
		buf,
		&gotgbot.SendPhotoOpts{ReplyToMessageId: ctx.EffectiveMessage.MessageId},
	)
	return err
}
