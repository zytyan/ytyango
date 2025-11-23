package myhandlers

import (
	"context"
	"fmt"
	g "main/globalcfg"
	"main/globalcfg/h"
	"main/helpers/azure"
	"sync"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
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

// saveNsfw
// param score: [0, 2, 4, 6]
func saveNsfw(fileUid, fileId string, severity int) {
	err := g.Q.AddPic(context.Background(), fileUid, fileId, severity)
	if err != nil {
		log.Warnf("save nsfw failed for fileId=%s err=%s", fileId, err)
	}
}

var nsfwReplyMsgList = [2][3]string{
	{"涩涩的~", "好色哦~", "口夷~"},
	{"给bot也看看~", "悄悄看一眼~", "不敢看~"},
}

func replyNsfw(bot *gotgbot.Bot, msg *gotgbot.Message, result *azure.ModeratorV2Result) (bool, error) {
	severity := result.GetSeverityByCategory(azure.ModerateV2CatSexual)
	if severity < 2 {
		return false, nil
	} else if severity > 7 {
		return false, fmt.Errorf("severity %d is invalid", severity)
	}
	photo := msg.Photo[len(msg.Photo)-1]

	go saveNsfw(photo.FileUniqueId, photo.FileId, severity)
	if severity >= 6 {
		g.Q.ChatStatToday(msg.Chat.Id).IncAdultCount()
	} else {
		g.Q.ChatStatToday(msg.Chat.Id).IncRacyCount()
	}
	var spoiler = 0
	if msg.HasMediaSpoiler {
		spoiler = 1
	}
	replyText := nsfwReplyMsgList[spoiler][severity/2-1]
	_, err := msg.Reply(bot, replyText, nil)
	return true, err
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

func NsfwDetect(bot *gotgbot.Bot, ctx *ext.Context) error {
	if ctx.Message.MediaGroupId != "" {
		moderateDetectGrouped(bot, ctx.Message)
		return nil
	}
	moderateDetectOne(bot, ctx.Message)
	return nil
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
	go saveNsfw(photo.FileUniqueId, photo.FileId, severity)
	text := fmt.Sprintf("score: %d", severity)
	_, err = ctx.Message.Reply(bot, text, nil)
	if err != nil {
		log.Warnf("reply message failed, err: %s", err)
	}
	return
}

func DetectNsfwPhoto(msg *gotgbot.Message) bool {
	if !HasImage(msg) {
		return false
	}
	if !h.ChatAutoCheckAdult(msg.Chat.Id) {
		return false
	}
	return true
}

func CountNsfwPics(bot *gotgbot.Bot, ctx *ext.Context) error {
	var racyPicCnt, adultPicCnt, manualNotNsfwCount int64
	text := fmt.Sprintf(
		"racy pic count: %d\n"+
			"adult pic count: %d\n"+
			"manual not nsfw pic count %d",
		racyPicCnt, adultPicCnt, manualNotNsfwCount)
	_, err := ctx.EffectiveMessage.Reply(bot, text, nil)
	return err
}
