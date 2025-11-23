package myhandlers

import (
	"context"
	"fmt"
	g "main/globalcfg"
	"main/globalcfg/h"
	"main/helpers/azure"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
)

const (
	nsfwCallbackButtonCmdScore = "s"
)

func BuildNsfwRateButton(fileUid, extraCmd string) *gotgbot.InlineKeyboardMarkup {
	cb := func(s int) string {
		return fmt.Sprintf("nsfw:%d:%s:%s", s, fileUid, extraCmd)
	}
	txt := func(s string, count int64) string {
		if count == 0 {
			return s
		}
		return fmt.Sprintf("%s (%d 用户评分)", s, count)
	}
	var rateTable [8]int64
	rateList, err := g.Q.GetPicRateDetailsByFileUid(context.Background(), fileUid)
	if err == nil {
		for _, rate := range rateList {
			if rate.Rating >= 8 || rate.Rating < 0 {
				continue
			}
			rateTable[rate.Rating] = rate.Count
		}
	}
	replyMarkup := h.NewInlineKeyboardButtonBuilder().
		Callback(txt("不色！", rateTable[0]), cb(0)).
		Callback(txt("有点涩", rateTable[2]), cb(2)).
		Row().
		Callback(txt("好色哦", rateTable[4]), cb(4)).
		Callback(txt("色爆了", rateTable[6]), cb(6)).
		Build()
	return replyMarkup
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
	replyMarkup := BuildNsfwRateButton(photo.FileUniqueId, "")

	replyText := nsfwReplyMsgList[spoiler][severity/2-1]
	_, err := msg.Reply(bot, replyText, &gotgbot.SendMessageOpts{
		ReplyMarkup: replyMarkup,
	})
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
	savedPic, err := g.Q.GetNsfwPicByFileUid(context.Background(), photo.FileUniqueId)
	userRate := severity
	if err == nil {
		userRate = int(savedPic.UserRate)
	}
	replyMarkup := BuildNsfwRateButton(photo.FileUniqueId, nsfwCallbackButtonCmdScore)
	text := fmt.Sprintf("bot评分: %d/6\n用户评分: %d/6", severity, userRate)
	_, err = ctx.Message.Reply(bot, text, &gotgbot.SendMessageOpts{
		ReplyMarkup: replyMarkup,
	})
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

var reUserRateInMsg = regexp.MustCompile(`用户评分.*(\d|\?\?\?)/6$`)

func refreshMsgFromBtn(bot *gotgbot.Bot, ctx *ext.Context, fileUid, cmd string) {
	msg := ctx.CallbackQuery.Message
	iMsg, ok := ctx.CallbackQuery.Message.(gotgbot.Message)

	if cmd == nsfwCallbackButtonCmdScore && ok {
		pic, err := g.Q.GetNsfwPicByFileUid(context.Background(), fileUid)
		if err != nil {
			log.Warnf("GetPicRateDetailsByFileUid failed, err: %s", err)
			return
		}
		userRate := float64(pic.UserRate)
		if pic.RateUserCount != 0 {
			userRate = float64(pic.UserRatingSum) / float64(pic.RateUserCount)
		}
		_, _, err = msg.EditText(bot,
			reUserRateInMsg.ReplaceAllString(iMsg.Text, fmt.Sprintf("用户评分: %.1f/6", userRate)),
			&gotgbot.EditMessageTextOpts{ReplyMarkup: *BuildNsfwRateButton(fileUid, cmd)})
		if err != nil {
			log.Warnf("reply message failed, err: %s", err)
		}
		return
	}
	opts := &gotgbot.EditMessageReplyMarkupOpts{
		ReplyMarkup: *BuildNsfwRateButton(fileUid, cmd),
	}
	_, _, err := msg.EditReplyMarkup(bot, opts)
	if err != nil {
		log.Warnf("edit message button failed, err: %s", err)
	}
	return
}

func RateNsfwPicByBtn(bot *gotgbot.Bot, ctx *ext.Context) error {
	cb := ctx.CallbackQuery
	split := strings.Split(cb.Data, ":")
	if len(split) < 3 {
		_, _ = cb.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{
			Text:      "Bot出现错误",
			ShowAlert: false,
		})
		return fmt.Errorf("invalid format callback %s", cb.Data)
	}
	_, r, fid := split[0], split[1], split[2]
	refresh := ""
	if len(split) > 3 && split[3] != "" {
		refresh = split[3]
	}
	defer refreshMsgFromBtn(bot, ctx, fid, refresh)
	rate := defaultAtoi(r, 0)
	rated, oldRate, err := g.Q.RatePic(context.Background(), fid, ctx.EffectiveSender.Id(), int64(rate))
	if err != nil {
		_, _ = cb.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{
			Text: "Bot出现错误，无法评分。",
		})
		return err
	}
	if !rated {
		_, err = cb.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{
			Text: "评分成功",
		})
		return err
	}
	if int64(rate) == oldRate {
		_, err = cb.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{
			Text:      "您不能重复评分",
			ShowAlert: true,
		})
		return err
	}
	_, err = cb.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{
		Text:      fmt.Sprintf("您的评分已由%d修改为%d", oldRate, rate),
		ShowAlert: true,
	})
	return err
}

func IsNsfwPicRateBtn(cb *gotgbot.CallbackQuery) bool {
	return strings.HasPrefix(cb.Data, "nsfw:")
}
