package myhandlers

import (
	"encoding/base64"
	"fmt"
	"hash/fnv"
	"html"
	"main/globalcfg"
	"main/helpers/bili"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
)

var log = globalcfg.GetLogger("handlers")

func BiliMsgFilter(msg *gotgbot.Message) bool {
	if !GetGroupInfo(msg.Chat.Id).AutoCvtBili {
		return false
	}
	if msg.ViaBot != nil || strings.HasPrefix(msg.Text, "/") {
		return false
	}
	return msg.Text != ""
}

func BiliMsgConverter(bot *gotgbot.Bot, ctx *ext.Context) (err error) {
	prepare, err := bili.ConvertBilibiliLinks(ctx.Message.Text)
	if err != nil || !prepare.NeedClean {
		// 没有B站链接
		return nil
	}
	log.Debugf("convert bilibili link %s", ctx.Message.Text)

	bv := html.EscapeString(prepare.BvText)
	text := fmt.Sprintf(`来自<a href="tg://user?id=%d">%s</a>的消息，其中的链接已自动转换%s%s`,
		ctx.EffectiveSender.Id(),
		ctx.EffectiveSender.Name(),
		"\n\n",
		bv)
	_, err = bot.SendMessage(ctx.Message.Chat.Id, text, &gotgbot.SendMessageOpts{ParseMode: `HTML`,
		ReplyMarkup: &gotgbot.InlineKeyboardMarkup{
			InlineKeyboard: [][]gotgbot.InlineKeyboardButton{{{
				Text:         "下载视频",
				CallbackData: biliCallbackData,
			}}},
		},
	})
	if err != nil {
		log.Infof("convert bilibili link send message %s failed, error %s", text, err)
		return err
	}
	_, err = ctx.Message.Delete(bot, nil)

	if err != nil {
		log.Infof("convert bilibili link delete original message %s failed, error %s", ctx.Message.Text, err)
	}
	return err
}

func buildQueryResult(title, text string, markups [][]gotgbot.InlineKeyboardButton) gotgbot.InlineQueryResult {
	fnvhash := fnv.New128a()
	_, _ = fnvhash.Write([]byte(title + text))
	bytes := fnvhash.Sum(nil)
	id := base64.URLEncoding.EncodeToString(bytes)
	return &gotgbot.InlineQueryResultArticle{
		Id:    id,
		Title: title,
		InputMessageContent: &gotgbot.InputTextMessageContent{
			MessageText: text,
		},
		ReplyMarkup: &gotgbot.InlineKeyboardMarkup{
			InlineKeyboard: markups,
		},
	}
}

type BiliInlineResult struct {
	Uid     int64 `gorm:"primaryKey"`
	Text    string
	ChatId  int64
	Message int64
}

func init() {
	err := globalcfg.GetDb().AutoMigrate(&BiliInlineResult{})
	if err != nil {
		panic(err)
	}
}

var startEpoch int64 = 1718035200000
var lastGeneratedMs = atomic.Int64{}
var idInThisMs = atomic.Int64{}

func GenerateUid() int64 {
	// generate uid auto increment
	now := time.Now().UnixMilli()
	t := (now - startEpoch) << 22
	last := lastGeneratedMs.Load()
	if last == t {
		id := idInThisMs.Add(1)
		return t | id
	}
	lastGeneratedMs.Store(t)
	idInThisMs.Store(0)
	return t
}

const biliCallbackData = "download:bilibili"

func BiliMsgConverterInline(bot *gotgbot.Bot, ctx *ext.Context) (err error) {
	if ctx.InlineQuery.Query == "" {
		_, err := ctx.InlineQuery.Answer(bot, []gotgbot.InlineQueryResult{
			buildQueryResult("这个家伙什么也没有说", "这个家伙什么也没有说", nil)}, nil)
		return err
	}
	log.Debugf("convert bilibili link %s", ctx.InlineQuery.Query)
	links, err := bili.ConvertBilibiliLinks(ctx.InlineQuery.Query)
	if err != nil || !links.CanConvert() {
		_, err := ctx.InlineQuery.Answer(bot,
			[]gotgbot.InlineQueryResult{
				buildQueryResult("没有检测到可转换的Bilibili链接或出现错误，点击可原样发送", ctx.InlineQuery.Query, nil)},
			nil)
		return err
	}

	uid := GenerateUid()
	callbackData := biliInlineCallbackPrefix + strconv.FormatInt(uid, 16)
	var btns [][]gotgbot.InlineKeyboardButton
	if bili.HasVideoLink(links.BvText) {
		btns = append(btns, []gotgbot.InlineKeyboardButton{{
			Text:         "下载视频",
			CallbackData: callbackData,
		}})
	}
	globalcfg.GetDb().Model(&BiliInlineResult{}).Create(&BiliInlineResult{
		Uid:    uid,
		Text:   links.BvText,
		ChatId: ctx.InlineQuery.From.Id,
	})
	var results []gotgbot.InlineQueryResult
	results = append(results, buildQueryResult("转换后的链接", links.BvText, btns))
	if links.HasAv {
		results = append(results, buildQueryResult("转换后的链接（使用AV号）", links.AvText, btns))
	}
	_, err = ctx.InlineQuery.Answer(bot, results, nil)

	return err
}
func getBiliCallbackDataInMsg(ctx *ext.Context) (uid int64, err error) {
	msg := ctx.EffectiveMessage
	if msg.ReplyMarkup == nil || len(msg.ReplyMarkup.InlineKeyboard) == 0 {
		return 0, fmt.Errorf("no inline keyboard")
	}
	for _, row := range msg.ReplyMarkup.InlineKeyboard {
		for _, btn := range row {
			if strings.HasPrefix(btn.CallbackData, biliInlineCallbackPrefix) {
				uid, err = strconv.ParseInt(btn.CallbackData[len(biliInlineCallbackPrefix):], 16, 64)
				return uid, err
			}
		}
	}
	return 0, fmt.Errorf("no bili inline button")
}
func SaveBiliMsgCallbackMsgId(_ *gotgbot.Bot, ctx *ext.Context) (err error) {
	uid, err := getBiliCallbackDataInMsg(ctx)
	if err != nil {
		return err
	}
	var result BiliInlineResult
	err = globalcfg.GetDb().Model(&BiliInlineResult{}).Where("uid = ?", uid).First(&result).Error
	if err != nil {
		return err
	}
	result.ChatId = ctx.EffectiveMessage.Chat.Id
	result.Message = ctx.EffectiveMessage.MessageId
	err = globalcfg.GetDb().Model(&BiliInlineResult{}).Where("uid = ?", uid).
		Omit("Uid Text").
		Updates(&result).Error
	return err
}
func IsBilibiliBtn(cq *gotgbot.CallbackQuery) bool {
	return cq.Data == biliCallbackData
}

const biliInlineCallbackPrefix = "il:bili:"

func IsBilibiliInlineBtn2(msg *gotgbot.Message) bool {
	if msg.ViaBot == nil || msg.ViaBot.Id != GetMainBot().Id {
		return false
	}
	if msg.ReplyMarkup == nil || len(msg.ReplyMarkup.InlineKeyboard) == 0 {
		return false
	}
	log.Infoln(msg.ReplyMarkup.InlineKeyboard)
	for _, row := range msg.ReplyMarkup.InlineKeyboard {
		for _, btn := range row {
			fmt.Println(btn.CallbackData)
			if strings.HasPrefix(btn.CallbackData, biliInlineCallbackPrefix) {
				return true
			}
		}
	}
	return false
}
func IsBilibiliInlineBtn(cq *gotgbot.CallbackQuery) bool {
	return strings.HasPrefix(cq.Data, biliInlineCallbackPrefix)
}
