package myhandlers

import (
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	jsoniter "github.com/json-iterator/go"
	"main/globalcfg"
	"net/http"
	"os"
)

func QrScan(bot *gotgbot.Bot, ctx *ext.Context) error {
	// 优先检查发的消息中有没有图片，然后检查回复的消息中有没有图片
	// 如果都没有图片，就不处理
	var photo *gotgbot.PhotoSize
	if ctx.Message.Photo != nil {
		photo = &ctx.Message.Photo[len(ctx.EffectiveMessage.Photo)-1]
	} else if ctx.Message.ReplyToMessage != nil && ctx.Message.ReplyToMessage.Photo != nil {
		photo = &ctx.Message.ReplyToMessage.Photo[len(ctx.EffectiveMessage.ReplyToMessage.Photo)-1]
	}
	if photo == nil {
		_, err := ctx.EffectiveMessage.Reply(bot, "bot 没有看到图片呢", nil)
		return err
	}
	file, err := bot.GetFile(photo.FileId, nil)
	if err != nil {
		return err
	}
	fp, err := os.Open(file.FilePath)
	if err != nil {
		return err
	}
	post, err := http.Post(globalcfg.GetConfig().QrScanUrl, "image/jpeg", fp)
	if err != nil {
		_, err = ctx.EffectiveMessage.Reply(bot, "bot解析QR出错了呢~", nil)
		return err
	}
	qrRes := QrRes{}
	err = jsoniter.NewDecoder(post.Body).Decode(&qrRes)
	if err != nil {
		_, err = ctx.EffectiveMessage.Reply(bot, "bot解析QR出错了呢~", nil)
		return err
	}
	if qrRes.Empty() {
		_, err = ctx.EffectiveMessage.Reply(bot, "bot没有识别到二维码呢~", nil)
		return err
	}
	_, err = ctx.EffectiveMessage.Reply(bot, "Bot检查到以下二维码：\n\n"+qrRes.ToString(), nil)
	return err
}
