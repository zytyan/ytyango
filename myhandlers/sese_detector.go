package myhandlers

import (
	"fmt"
	"main/globalcfg"
	"main/groupstatv2"
	"main/helpers/azure"
	"strings"
	"sync"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"gorm.io/gorm/clause"
)

type groupedMsgK struct {
	ChatId  int64
	GroupId string
}

func HasImage(msg *gotgbot.Message) bool {
	if msg == nil {
		return false
	}
	if len(msg.Photo) == 0 {
		return false
	}
	return true
}

type NsfwPicRacy struct {
	PicId string `gorm:"uniqueIndex"`
}
type NsfwPicAdult struct {
	PicId string `gorm:"uniqueIndex"`
}

type ManualNotNsfwPicAdult struct {
	PicId string `gorm:"uniqueIndex"`
}

type ManualNotNsfwPicRacy struct {
	PicId string `gorm:"uniqueIndex"`
}

func saveNsfw(picId string, isRacy, isAdult bool) {
	log.Debugf("picId = %s, isRacy = %t, isAdult = %t", picId, isRacy, isAdult)
	if !isAdult && !isRacy {
		return
	}
	if isAdult {
		if globalcfg.GetDb().Where("pic_id = ?", picId).First(&ManualNotNsfwPicAdult{PicId: picId}).Error == nil {
			log.Debugf("Image %s in Manul not nsfw pic adult.", picId)
			return
		}
		globalcfg.GetDb().Clauses(
			clause.OnConflict{
				DoNothing: true,
			},
		).Create(&NsfwPicAdult{
			PicId: picId,
		})
	} else {
		if globalcfg.GetDb().Where("pic_id = ?", picId).First(&ManualNotNsfwPicRacy{PicId: picId}).Error == nil {
			log.Debugf("Image %s in Manul not nsfw pic racy.", picId)
			return
		}
		globalcfg.GetDb().Clauses(
			clause.OnConflict{
				DoNothing: true,
			},
		).Create(&NsfwPicRacy{
			PicId: picId,
		})
	}
}

func replyNsfw(bot *gotgbot.Bot, msg *gotgbot.Message, result *azure.ModeratorV2Result) (bool, error) {
	severity := result.GetSeverityByCategory(azure.ModerateV2CatSexual)
	isAdult := severity >= 6
	isRacy := severity >= 4
	if !isAdult && !isRacy {
		return false, nil
	}
	go saveNsfw(msg.Photo[len(msg.Photo)-1].FileId, isRacy, isAdult)
	if isRacy {
		groupstatv2.GetGroupToday(msg.Chat.Id).RacyCount.Inc()
	}
	if isAdult {
		groupstatv2.GetGroupToday(msg.Chat.Id).AdultCount.Inc()
	}



	if !msg.HasMediaSpoiler {
		if isAdult {
			_, err := msg.Reply(bot, "口夷~", nil)
			return true, err
		} else if isRacy {
			_, err := msg.Reply(bot, "好色哦~", nil)
			return true, err
		} else {
			return false, nil
		}
	} else {
		if isAdult {
			_, err := msg.Reply(bot, "不敢看~", nil)
			return true, err
		} else if isRacy {
			_, err := msg.Reply(bot, "悄悄看一眼~", nil)
			return true, err
		} else {
			return false, nil
		}
	}
}

func moderateDetectOne(bot *gotgbot.Bot, msg *gotgbot.Message) (replied bool) {
	moderatorResult, err := moderatorMsg(bot, &msg.Photo[len(msg.Photo)-1])
	if err != nil {
		log.Warnf("moderate msg failed, err: %s", err)
		return
	}
	replied, err = replyNsfw(bot, msg, moderatorResult)
	if err != nil {
		log.Warnf("reply message failed, err: %s", err)
	}
	return
}

var groupedDetectMap = &sync.Map{}

func moderateDetectGrouped(bot *gotgbot.Bot, msg *gotgbot.Message) {
	key := groupedMsgK{ChatId: msg.Chat.Id, GroupId: msg.MediaGroupId}
	chn := make(chan *gotgbot.Message, 10)
	existsChn, ok := groupedDetectMap.LoadOrStore(key, chn)
	if ok {
		close(chn)
		existsChn.(chan *gotgbot.Message) <- msg
		return
	}
	defer groupedDetectMap.Delete(key)
	for {
		select {
		case <-time.After(10 * time.Second):
			groupedDetectMap.Delete(key)
			close(chn)
			return
		case msg, ok := <-chn:
			if !ok {
				return
			}
			if moderateDetectOne(bot, msg) {
				return
			}
		}
	}

}

func seseDetect(bot *gotgbot.Bot, ctx *ext.Context) {
	if !HasImage(ctx.Message) {
		return
	}
	if ctx.Message.MediaGroupId != "" {
		moderateDetectGrouped(bot, ctx.Message)
		return
	}
	moderateDetectOne(bot, ctx.Message)
}

func CmdScore(bot *gotgbot.Bot, ctx *ext.Context) (err error) {
	var photo *gotgbot.PhotoSize
	if len(ctx.Message.Photo) != 0 {
		photo = &ctx.Message.Photo[len(ctx.Message.Photo)-1]
	} else if ctx.Message.ReplyToMessage != nil && len(ctx.Message.ReplyToMessage.Photo) != 0 {
		photo = &ctx.Message.ReplyToMessage.Photo[len(ctx.Message.ReplyToMessage.Photo)-1]
	} else {
		_, err := ctx.Message.Reply(bot, "没有图片", nil)
		if err != nil {
			log.Warnf("reply message failed, err: %s", err)
		}
		return err
	}
	result, err := moderatorMsg(bot, photo)
	if err != nil {
		_, err := ctx.Message.Reply(bot, "识别失败", nil)
		if err != nil {
			log.Warnf("reply message failed, err: %s", err)
		}
		return err
	}
	severity := result.GetSeverityByCategory(azure.ModerateV2CatSexual)
	go saveNsfw(photo.FileId, severity >= 4, severity >= 6)
	text := fmt.Sprintf("score: %d", severity)
	_, err = ctx.Message.Reply(bot, text, nil)
	if err != nil {
		log.Warnf("reply message failed, err: %s", err)
	}
	return
}

func SafeGo(f func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Errorf("panic in SafeGo: %s", r)
			}
		}()
		f()
	}()
}

func SeseDetect(bot *gotgbot.Bot, ctx *ext.Context) error {
	groupInfo := GetGroupInfo(ctx.Message.Chat.Id)
	if groupInfo != nil && !groupInfo.AutoCheckAdult {
		return nil
	}
	SafeGo(func() { seseDetect(bot, ctx) })
	return nil
}

type manualAddPicCfg struct {
	addingRacy  bool
	addingAdult bool
}

var gAddPicCfg manualAddPicCfg

func HasPhoto(message *gotgbot.Message) bool {
	photos := message.Photo
	if len(photos) == 0 {
		return false
	}
	return true
}
func SetManualAddPic(bot *gotgbot.Bot, ctx *ext.Context) error {
	lower := strings.ToLower(ctx.EffectiveMessage.Text)
	if strings.Contains(lower, "adult") {
		gAddPicCfg.addingAdult = true
		gAddPicCfg.addingRacy = false
	} else if strings.Contains(lower, "racy") {
		gAddPicCfg.addingAdult = false
		gAddPicCfg.addingRacy = true
	} else {
		gAddPicCfg.addingAdult = false
		gAddPicCfg.addingRacy = false
	}
	_, err := ctx.EffectiveMessage.Reply(bot, fmt.Sprintf("%#v", gAddPicCfg), nil)
	return err
}

func ManualAddPic(_ *gotgbot.Bot, ctx *ext.Context) error {
	if ctx.EffectiveUser.Id != globalcfg.GetConfig().God {
		return nil
	}
	photos := ctx.EffectiveMessage.Photo
	photo := photos[len(photos)-1]
	saveNsfw(photo.FileId, gAddPicCfg.addingRacy, gAddPicCfg.addingAdult)
	return nil
}
func CountNsfwPics(bot *gotgbot.Bot, ctx *ext.Context) error {
	var racyPicCnt, adultPicCnt, manualNotRacyCnt, manualNotAdultCnt int64
	globalcfg.GetDb().Model(NsfwPicRacy{}).Count(&racyPicCnt)
	globalcfg.GetDb().Model(NsfwPicAdult{}).Count(&adultPicCnt)

	globalcfg.GetDb().Model(ManualNotNsfwPicRacy{}).Count(&manualNotRacyCnt)
	globalcfg.GetDb().Model(ManualNotNsfwPicAdult{}).Count(&manualNotAdultCnt)
	text := fmt.Sprintf(
		"racy pic count: %d\n"+
			"adult pic count: %d\n"+
			"manual not racy pic count: %d\n"+
			"manual not adult pic count %d",
		racyPicCnt, adultPicCnt, manualNotRacyCnt, manualNotAdultCnt)
	_, err := ctx.EffectiveMessage.Reply(bot, text, nil)
	return err
}
