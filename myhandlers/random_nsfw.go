package myhandlers

import (
	"main/globalcfg"
	"regexp"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"gorm.io/gorm/clause"
)

func getRandomNsfwAdult() string {
	var result NsfwPicAdult
	globalcfg.GetDb().Order("RANDOM()").First(&result)
	return result.PicId
}
func getRandomNsfwRacy() string {
	var result NsfwPicRacy
	globalcfg.GetDb().Order("RANDOM()").First(&result)
	return result.PicId
}

func SendRandAdult(bot *gotgbot.Bot, ctx *ext.Context) error {
	photo := getRandomNsfwAdult()
	if photo == "" {
		_, err := ctx.EffectiveMessage.Reply(bot, "没有色图~", nil)
		return err
	}
	_, err := bot.SendPhoto(ctx.EffectiveChat.Id, gotgbot.InputFileByID(photo), &gotgbot.SendPhotoOpts{
		ReplyParameters: MakeReplyToMsgID(ctx.EffectiveMessage.MessageId),
		HasSpoiler:      true,
	})
	return err
}

func SendRandRacy(bot *gotgbot.Bot, ctx *ext.Context) error {
	photo := getRandomNsfwRacy()
	if photo == "" {
		_, err := ctx.EffectiveMessage.Reply(bot, "没有涩图~", nil)
		return err
	}
	_, err := bot.SendPhoto(ctx.EffectiveChat.Id, gotgbot.InputFileByID(photo), &gotgbot.SendPhotoOpts{
		ReplyParameters: MakeReplyToMsgID(ctx.EffectiveMessage.MessageId),
	})
	return err
}

var reRacyPattern = regexp.MustCompile(`来[张点个]([涩色瑟]|(se))图|再来一张|來[張點個]([澀色瑟]|(se))圖`)

//var reAdultPattern = regexp.MustCompile(`来[张点个]色图|來[張點個]色圖`)

func IsRequiredAdult(msg *gotgbot.Message) bool {
	if len(msg.Text) == 0 {
		return false
	}
	if !GetGroupInfo(msg.Chat.Id).AutoCheckAdult {
		return false
	}
	return false
}

func IsRequiredRacy(msg *gotgbot.Message) bool {
	if len(msg.Text) == 0 {
		return false
	}
	if !GetGroupInfo(msg.Chat.Id).AutoCheckAdult {
		return false
	}
	return reRacyPattern.MatchString(msg.Text)
}

func markPicNotAdult(picId string) {
	globalcfg.GetDb().Clauses(
		clause.OnConflict{
			DoNothing: true,
		},
	).Create(&ManualNotNsfwPicAdult{
		PicId: picId,
	})
	log.Infof("markPicNotAdult %s", picId)
	globalcfg.GetDb().Where("pic_id = ?", picId).Delete(
		&NsfwPicAdult{PicId: picId},
	)
}
func markPicIsAdult(picId string) {
	globalcfg.GetDb().Clauses(
		clause.OnConflict{
			DoNothing: true,
		},
	).Create(&NsfwPicAdult{
		PicId: picId,
	})
	log.Infof("markPicIsAdult %s", picId)
	globalcfg.GetDb().Where("pic_id = ?", picId).Delete(
		&ManualNotNsfwPicAdult{PicId: picId},
	)
}
func markPicNotRacy(picId string) {
	globalcfg.GetDb().Clauses(
		clause.OnConflict{
			DoNothing: true,
		},
	).Create(&ManualNotNsfwPicRacy{
		PicId: picId,
	})
	log.Infof("markPicNotRacy %s", picId)
	globalcfg.GetDb().Where("pic_id = ?", picId).Delete(
		&NsfwPicRacy{PicId: picId},
	)
}
func markPicIsRacy(picId string) {
	globalcfg.GetDb().Clauses(
		clause.OnConflict{
			DoNothing: true,
		},
	).Create(&NsfwPicRacy{
		PicId: picId,
	})
	log.Infof("markPicIsRacy %s", picId)
	globalcfg.GetDb().Where("pic_id = ?", picId).Delete(
		&ManualNotNsfwPicRacy{PicId: picId},
	)
}

func MarkPicNotAdult(bot *gotgbot.Bot, ctx *ext.Context) error {
	var photo gotgbot.PhotoSize
	if ctx.EffectiveMessage.Photo != nil {
		photo = ctx.EffectiveMessage.Photo[len(ctx.EffectiveMessage.Photo)-1]
	} else if ctx.EffectiveMessage.ReplyToMessage != nil && ctx.EffectiveMessage.ReplyToMessage.Photo != nil {
		photo = ctx.EffectiveMessage.ReplyToMessage.Photo[len(ctx.EffectiveMessage.ReplyToMessage.Photo)-1]
	} else {
		return nil
	}
	picId := photo.FileId
	markPicNotAdult(picId)
	_, err := ctx.EffectiveMessage.Reply(bot, "已标记为非色图", nil)
	return err
}

func MarkPicNotRacy(bot *gotgbot.Bot, ctx *ext.Context) error {
	var photo gotgbot.PhotoSize
	if ctx.EffectiveMessage.Photo != nil {
		photo = ctx.EffectiveMessage.Photo[len(ctx.EffectiveMessage.Photo)-1]
	} else if ctx.EffectiveMessage.ReplyToMessage != nil && ctx.EffectiveMessage.ReplyToMessage.Photo != nil {
		photo = ctx.EffectiveMessage.ReplyToMessage.Photo[len(ctx.EffectiveMessage.ReplyToMessage.Photo)-1]
	} else {
		return nil
	}
	picId := photo.FileId
	markPicNotRacy(picId)
	_, err := ctx.EffectiveMessage.Reply(bot, "已标记为非涩图", nil)
	return err
}

func MarkPicIsAdult(bot *gotgbot.Bot, ctx *ext.Context) error {
	var photo gotgbot.PhotoSize
	if ctx.EffectiveMessage.Photo != nil {
		photo = ctx.EffectiveMessage.Photo[len(ctx.EffectiveMessage.Photo)-1]
	} else if ctx.EffectiveMessage.ReplyToMessage != nil && ctx.EffectiveMessage.ReplyToMessage.Photo != nil {
		photo = ctx.EffectiveMessage.ReplyToMessage.Photo[len(ctx.EffectiveMessage.ReplyToMessage.Photo)-1]
	} else {
		return nil
	}
	picId := photo.FileId
	markPicIsAdult(picId)
	_, err := ctx.EffectiveMessage.Reply(bot, "已标记为色图", nil)
	return err
}

func MarkPicIsRacy(bot *gotgbot.Bot, ctx *ext.Context) error {
	var photo gotgbot.PhotoSize
	if ctx.EffectiveMessage.Photo != nil {
		photo = ctx.EffectiveMessage.Photo[len(ctx.EffectiveMessage.Photo)-1]
	} else if ctx.EffectiveMessage.ReplyToMessage != nil && ctx.EffectiveMessage.ReplyToMessage.Photo != nil {
		photo = ctx.EffectiveMessage.ReplyToMessage.Photo[len(ctx.EffectiveMessage.ReplyToMessage.Photo)-1]
	} else {
		return nil
	}
	picId := photo.FileId
	markPicIsRacy(picId)
	_, err := ctx.EffectiveMessage.Reply(bot, "已标记为涩图", nil)
	return err
}
