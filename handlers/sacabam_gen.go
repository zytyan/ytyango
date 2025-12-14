package handlers

import (
	"bytes"
	"errors"
	"fmt"
	"main/helpers/imgproc"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/disintegration/imaging"
)

func IsSacabam(msg *gotgbot.Message) bool {
	return imgproc.MatchSacabambaspis(msg.Text)
}

func GenSacabam(bot *gotgbot.Bot, ctx *ext.Context) error {
	img, err := imgproc.GenSacaImage(ctx.EffectiveMessage.Text)
	if err != nil {
		var e *imgproc.ErrTooLongSacaList
		if errors.As(err, &e) {
			text := fmt.Sprintf("要拼接%d个图，太长了，bot最多只能拼接%d个", e.Len, e.Limit)
			_, err = ctx.EffectiveMessage.Reply(bot, text, nil)
			return err
		}
		log.Warnf("genSacabam err: %v", err)
		return err
	}
	buf := &bytes.Buffer{}
	err = imaging.Encode(buf, img, imaging.JPEG)
	if err != nil {
		return err
	}
	_, err = bot.SendPhoto(ctx.EffectiveChat.Id,
		gotgbot.InputFileByReader("sacabam.jpg", buf),
		&gotgbot.SendPhotoOpts{
			ReplyParameters: &gotgbot.ReplyParameters{
				MessageId: ctx.EffectiveMessage.MessageId,
			},
		},
	)
	return err
}
