package myhandlers

import (
	"main/globalcfg/h"
	"os"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/disintegration/imaging"
	"github.com/kolesa-team/go-webp/webp"
)

func webp2png(file string) (string, error) {
	f, err := os.Open(file)
	if err != nil {
		return "", err
	}
	defer f.Close()
	img, err := webp.Decode(f, nil)
	if err != nil {
		return "", err
	}
	out, err := os.CreateTemp(os.TempDir(), "webp2png*.png")
	if err != nil {
		return "", err
	}
	defer out.Close()
	err = imaging.Encode(out, img, imaging.PNG)
	return out.Name(), err
}

func WebpToPng(bot *gotgbot.Bot, ctx *ext.Context) error {
	if ctx.Message == nil {
		return nil
	}
	if ctx.Message.ReplyToMessage == nil || ctx.Message.ReplyToMessage.Sticker == nil {
		_, err := ctx.Message.Reply(bot, ""+
			"本功能用于解决在Telegram中发送Webp图片无法被正确发送，而是变为一个sticker的问题。\n"+
			"理论上也可以用于将Webp下载为Png格式，但这个功能并非为此开发，不接受对此的bug反馈。\n"+
			"需要回复一个webp图片，在Telegram中应该表现为一个表情包，但无法点开。", nil)
		return err
	}
	sticker := ctx.Message.ReplyToMessage.Sticker
	f, err := bot.GetFile(sticker.FileId, nil)
	if err != nil {
		return err
	}
	pngFile, err := webp2png(f.FilePath)
	if err != nil {
		_, err = ctx.Message.Reply(bot, err.Error(), nil)
		return err
	}
	msg := ctx.Message
	defer os.Remove(pngFile)
	_, err = bot.SendDocument(msg.Chat.Id, h.LocalFile(pngFile), &gotgbot.SendDocumentOpts{
		ReplyParameters: MakeReplyToMsgID(msg.Chat.Id),
	})
	return err
}
