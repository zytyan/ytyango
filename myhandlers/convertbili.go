package myhandlers

import (
	"hash/fnv"
	"main/globalcfg"
	"strings"
)
import (
	"encoding/base64"
	"fmt"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"html"
	"main/helpers/bili"
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
	prepare, err := bili.ContainsBiliLinkAndTryPrepare(ctx.Message.Text)
	if err != nil {
		// 没有B站链接
		return nil
	}
	log.Debugf("convert bilibili link %s", ctx.Message.Text)
	if !prepare.NeedConvert() {
		return nil
	}
	bv, err := prepare.ToBv()
	if err != nil {
		log.Infof("convert bilibili link %s failed, error %s", ctx.Message.Text, err)
		return err
	}

	bv = html.EscapeString(bv)
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

const biliCallbackData = "download:bilibili"

func BiliMsgConverterInline(bot *gotgbot.Bot, ctx *ext.Context) (err error) {
	if ctx.InlineQuery.Query == "" {
		_, err := ctx.InlineQuery.Answer(bot, []gotgbot.InlineQueryResult{
			buildQueryResult("这个家伙什么也没有说", "这个家伙什么也没有说", nil)}, nil)
		return err
	}
	log.Debugf("convert bilibili link %s", ctx.InlineQuery.Query)
	prepare, err := bili.ContainsBiliLinkAndTryPrepare(ctx.InlineQuery.Query)
	if err != nil || !prepare.NeedConvert() {
		_, err := ctx.InlineQuery.Answer(bot,
			[]gotgbot.InlineQueryResult{
				buildQueryResult("没有检测到可转换的Bilibili链接，点击可原样发送", ctx.InlineQuery.Query, nil)},
			nil)
		return err
	}
	bv, err := prepare.ToBv()
	if err != nil {
		return err
	}
	_, err = ctx.InlineQuery.Answer(bot,
		[]gotgbot.InlineQueryResult{
			buildQueryResult("转换后的链接", bv, nil),
		}, nil)
	return err
}
func IsBilibiliBtn(cq *gotgbot.CallbackQuery) bool {
	return cq.Data == biliCallbackData
}
