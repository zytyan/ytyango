package myhandlers

import (
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"main/globalcfg"
	"main/helpers/imgproc"
	"os"
	"path/filepath"
)

type prprCache struct {
	ProfilePhotoUid string `gorm:"unique"`
	PrprFileId      string
}

func getPrprCache(profilePhotoFileId string) (string, error) {
	var cache prprCache
	err := globalcfg.GetDb().Where(&prprCache{ProfilePhotoUid: profilePhotoFileId}).First(&cache).Error
	if err != nil {
		return "", err
	}
	return cache.PrprFileId, nil
}

func setPrprCache(profilePhotoUid, prprFileId string) error {
	var cache prprCache
	cache.ProfilePhotoUid = profilePhotoUid
	cache.PrprFileId = prprFileId
	return globalcfg.GetDb().Create(&cache).Error
}

func GenPrpr(bot *gotgbot.Bot, ctx *ext.Context) (err error) {
	var userID int64
	userID = ctx.EffectiveUser.Id
	if ctx.Message.ReplyToMessage != nil {
		userID = ctx.Message.ReplyToMessage.From.Id
	}
	if userID == 0 {
		return
	}
	photos, err := bot.GetUserProfilePhotos(userID, nil)
	if err != nil {
		return err
	}
	if len(photos.Photos) == 0 {
		_, err = ctx.Message.Reply(bot, "没有头像", nil)
		return
	}
	photo := photos.Photos[0][0]
	prpr, err := getPrprCache(photo.FileUniqueId)
	if err == nil {
		_, err = bot.SendSticker(ctx.EffectiveChat.Id, prpr, nil)
		return err
	}
	file, err := bot.GetFile(photo.FileId, nil)
	if err != nil {
		return err
	}
	filePath := filepath.Join(os.TempDir(), photo.FileUniqueId+"_prpr.webp")
	filePath, err = filepath.Abs(filePath)
	if err != nil {
		return err
	}
	defer os.Remove(filePath)
	err = imgproc.GenPrpr(file.FilePath, filePath)
	if err != nil {
		return err
	}
	send, err := bot.SendSticker(ctx.EffectiveChat.Id, fileSchema(filePath), nil)
	if err != nil {
		return err
	}
	err = setPrprCache(photo.FileUniqueId, send.Sticker.FileId)
	return err
}
