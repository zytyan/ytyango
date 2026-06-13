package handlers

import (
	"context"
	"main/globalcfg"
	"main/globalcfg/h"
	"main/helpers/imgproc"
	"os"
	"path/filepath"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
)

func GenPrpr(bot *gotgbot.Bot, ctx *ext.Context) (err error) {
	msg := ctx.Message
	if msg == nil || ctx.EffectiveChat == nil || ctx.EffectiveUser == nil {
		return nil
	}
	userID := ctx.EffectiveUser.Id
	if msg.ReplyToMessage != nil && msg.ReplyToMessage.From != nil {
		userID = msg.ReplyToMessage.From.Id
	}
	if userID == 0 {
		return
	}
	photos, err := bot.GetUserProfilePhotos(userID, nil)
	if err != nil {
		return err
	}
	if len(photos.Photos) == 0 || len(photos.Photos[0]) == 0 {
		_, err = msg.Reply(bot, "没有头像", nil)
		return
	}
	photo := photos.Photos[0][0]
	prpr, err := g.Q.GetPrprCache(context.Background(), photo.FileUniqueId)
	if err == nil {
		_, err = bot.SendSticker(ctx.EffectiveChat.Id, gotgbot.InputFileByID(prpr), nil)
		return err
	}
	file, err := h.DownloadToDisk(bot, photo.FileId)
	if err != nil {
		return err
	}
	filePath := filepath.Join(os.TempDir(), photo.FileUniqueId+"_prpr.webp")
	filePath, err = filepath.Abs(filePath)
	if err != nil {
		return err
	}
	defer os.Remove(filePath)
	err = imgproc.GenPrpr(file, filePath)
	if err != nil {
		return err
	}
	send, err := bot.SendSticker(ctx.EffectiveChat.Id, h.LocalFile(filePath), nil)
	if err != nil {
		return err
	}
	err = g.Q.SetPrprCache(context.Background(), photo.FileUniqueId, send.Sticker.FileId)
	return err
}
