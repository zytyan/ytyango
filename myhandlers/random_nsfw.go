package myhandlers

import (
	"fmt"
	"main/globalcfg"
	"regexp"
	"strings"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
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

func IsRequiredAdult(msg *gotgbot.Message) bool {
	if len(msg.Text) == 0 {
		return false
	}
	if !GetGroupInfo(msg.Chat.Id).RespNsfwMsg {
		return false
	}
	return false
}

func IsRequiredRacy(msg *gotgbot.Message) bool {
	if len(msg.Text) == 0 {
		return false
	}
	if !GetGroupInfo(msg.Chat.Id).RespNsfwMsg {
		return false
	}
	return reRacyPattern.MatchString(msg.Text)
}

type NotNsfwPic struct {
	PicId string `gorm:"uniqueIndex"`
}

func markPicNotNsfw(bot *gotgbot.Bot, msg *gotgbot.Message, picId string) error {
	err := globalcfg.GetDb().Exec(`INSERT OR IGNORE INTO not_nsfw_pics (pic_id) VALUES (?)`, picId).Error
	err2 := globalcfg.GetDb().Exec(`DELETE FROM nsfw_pic_racies WHERE pic_id = ?`, picId).Error
	err3 := globalcfg.GetDb().Exec(`DELETE FROM nsfw_pic_adults WHERE pic_id = ?`, picId).Error

	if err != nil || err2 != nil || err3 != nil {
		_, err = msg.Reply(bot, fmt.Sprintf("error: %s, error2: %s, error3: %s", err, err2, err3), nil)
	} else {
		_, err = msg.Reply(bot, "已标记为非色图", nil)
	}
	return err
}

func removePicNotNsfwMark(bot *gotgbot.Bot, msg *gotgbot.Message, picId string) error {
	err := globalcfg.GetDb().Exec(`DELETE FROM not_nsfw_pics WHERE pic_id = ?`, picId).Error
	if err != nil {
		_, err = msg.Reply(bot, fmt.Sprintf("error: %s", err), nil)
	} else {
		_, err = msg.Reply(bot, "已移除非色图标记", nil)
	}
	return err
}

func MarkPicNotNsfwOrNot(bot *gotgbot.Bot, ctx *ext.Context) error {
	var photo gotgbot.PhotoSize
	if ctx.EffectiveMessage.Photo != nil {
		photo = ctx.EffectiveMessage.Photo[len(ctx.EffectiveMessage.Photo)-1]
	} else if ctx.EffectiveMessage.ReplyToMessage != nil && ctx.EffectiveMessage.ReplyToMessage.Photo != nil {
		photo = ctx.EffectiveMessage.ReplyToMessage.Photo[len(ctx.EffectiveMessage.ReplyToMessage.Photo)-1]
	} else {
		_, err := ctx.EffectiveMessage.Reply(bot, "没有找到图片呢~", nil)
		return err
	}
	text := getText(ctx)
	if strings.HasPrefix(text, "/remove_nsfw_mark") {
		return removePicNotNsfwMark(bot, ctx.EffectiveMessage, photo.FileId)
	}
	return markPicNotNsfw(bot, ctx.EffectiveMessage, photo.FileId)

}
