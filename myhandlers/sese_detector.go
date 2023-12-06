package myhandlers

import (
	"fmt"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"main/helpers/azure"
	"sync"
	"time"
)

func bool2yn(b bool) string {
	if b {
		return "Y"
	}
	return "N"
}

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

func replyNsfw(bot *gotgbot.Bot, msg *gotgbot.Message, result *azure.ModeratorResult, groupInfo *GroupInfo) (bool, error) {
	WithGroupLockToday(msg.Chat.Id, func(g *GroupStatDaily) {
		if groupInfo.ModeratorConfig.IsAdult(result) {
			g.AdultCount++
		} else if groupInfo.ModeratorConfig.IsRacy(result) {
			g.RacyCount++
		}
	})
	if !msg.HasMediaSpoiler {
		if groupInfo.ModeratorConfig.IsAdult(result) {
			_, err := msg.Reply(bot, "口夷~", nil)
			return true, err
		} else if groupInfo.ModeratorConfig.IsRacy(result) {
			_, err := msg.Reply(bot, "好色哦~", nil)
			return true, err
		} else {
			return false, nil
		}
	} else {
		if groupInfo.ModeratorConfig.IsAdult(result) {
			_, err := msg.Reply(bot, "不敢看~", nil)
			return true, err
		} else if groupInfo.ModeratorConfig.IsRacy(result) {
			_, err := msg.Reply(bot, "悄悄看一眼~", nil)
			return true, err
		} else {
			return false, nil
		}
	}
}

func moderateDetectOne(bot *gotgbot.Bot, msg *gotgbot.Message, groupInfo *GroupInfo) (replied bool) {
	moderatorResult, err := moderatorMsg(bot, &msg.Photo[len(msg.Photo)-1])
	if err != nil {
		log.Warnf("moderate msg failed, err: %s", err)
		return
	}
	replied, err = replyNsfw(bot, msg, moderatorResult, groupInfo)
	if err != nil {
		log.Warnf("reply message failed, err: %s", err)
	}
	return
}

var groupedDetectMap = &sync.Map{}

func moderateDetectGrouped(bot *gotgbot.Bot, msg *gotgbot.Message, groupInfo *GroupInfo) {
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
			if moderateDetectOne(bot, msg, groupInfo) {
				return
			}
		}
	}

}

func seseDetect(bot *gotgbot.Bot, ctx *ext.Context, groupInfo *GroupInfo) {
	if !HasImage(ctx.Message) {
		return
	}
	if ctx.Message.MediaGroupId != "" {
		moderateDetectGrouped(bot, ctx.Message, groupInfo)
		return
	}
	moderateDetectOne(bot, ctx.Message, groupInfo)
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
	text := fmt.Sprintf("audlt: %f [%s]\nracy: %f [%s]",
		result.AdultClassificationScore, bool2yn(result.IsImageAdultClassified),
		result.RacyClassificationScore, bool2yn(result.IsImageRacyClassified),
	)
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
	if !groupInfo.AutoCheckAdult {
		return nil
	}
	SafeGo(func() { seseDetect(bot, ctx, groupInfo) })
	return nil
}
