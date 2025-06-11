package myhandlers

import (
	"fmt"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"main/globalcfg"
	"net/url"
	"path/filepath"
	"sync"
	"time"
)

func fileSchema(filename string) string {
	if !filepath.IsAbs(filename) {
		var err error
		filename, err = filepath.Abs(filename)
		if err != nil {
			log.Error(err)
			return ""
		}
	}
	return "file://" + url.PathEscape(filename)
}

var mainBot *gotgbot.Bot

func SetMainBot(bot *gotgbot.Bot) {
	mainBot = bot
}

func GetMainBot() *gotgbot.Bot {
	return mainBot
}

func init() {
	db := globalcfg.GetDb()
	err := db.AutoMigrate(&User{}, &GroupInfo{}, &prprCache{}, &YtDlResult{}, &CharacterAttr{}, &NsfwPicRacy{}, &NsfwPicAdult{}, &ManualNotNsfwPicAdult{}, &ManualNotNsfwPicRacy{})
	if err != nil {
		panic(err)
	}
}

func GetMsgInfo(bot *gotgbot.Bot, ctx *ext.Context) error {
	data := fmt.Sprintf("获取消息信息：%d", ctx.EffectiveMessage.Chat.Id)
	_, err := ctx.EffectiveMessage.Reply(bot, data, nil)
	return err
}
func EnableCalc(bot *gotgbot.Bot, ctx *ext.Context) error {
	groupInfo := GetGroupInfo(ctx.EffectiveChat.Id)
	groupInfo.AutoCalculate = true
	globalcfg.GetDb().Save(&groupInfo)
	_, err := ctx.EffectiveMessage.Reply(bot, "已开启计算器", nil)
	return err
}

func cutString(s string, length int) string {
	rl := []rune(s)
	if len(rl) <= length {
		return s
	}
	return string(rl[:length-3]) + "..."
}

func MakeDebounceReply(bot *gotgbot.Bot, ctx *ext.Context, interval time.Duration) (func(s string) (*gotgbot.Message, error), func() error) {
	var l sync.Mutex
	var timer *time.Timer
	var sent *gotgbot.Message
	var err error
	return func(s string) (*gotgbot.Message, error) {
			l.Lock()
			defer l.Unlock()

			if timer != nil {
				timer.Stop()
			}
			timer = time.AfterFunc(interval, func() {
				l.Lock()
				defer l.Unlock()
				if timer != nil {
					timer.Stop()
				}
				timer = nil
				log.Infof("debounce reply %s", s)
				if sent == nil {
					sent, err = ctx.EffectiveMessage.Reply(bot, s, nil)
				} else {
					_, _, err = sent.EditText(bot, s, nil)
				}
			})
			return sent, err
		}, func() error {
			l.Lock()
			defer l.Unlock()
			if timer != nil {
				timer.Stop()
			}
			timer = nil
			if sent != nil {
				_, err = sent.Delete(bot, nil)
			}
			return err
		}
}

func MakeDebounceMustReply(bot *gotgbot.Bot, ctx *ext.Context, interval time.Duration) (func(s string) *gotgbot.Message, func()) {
	f, del := MakeDebounceReply(bot, ctx, interval)
	return func(s string) *gotgbot.Message {
			m, err := f(s)
			if err != nil {
				panic(err)
			}
			return m
		}, func() {
			_ = del()
		}
}

func MakeAnswerCallback(bot *gotgbot.Bot, ctx *ext.Context) func(string, bool) {
	l := sync.Mutex{}
	answerRan := false
	return func(text string, alert bool) {
		l.Lock()
		defer l.Unlock()
		if answerRan {
			log.Warnf("answer %d[%s] callback ran twice", ctx.CallbackQuery.Id, ctx.CallbackQuery.Data)
			return
		}
		answerRan = true
		log.Infof("answer %s[%s] callback %s", ctx.CallbackQuery.Id, ctx.CallbackQuery.Data, text)
		_, err := ctx.CallbackQuery.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{
			Text:      text,
			ShowAlert: alert,
		})
		if err != nil {
			log.Error(err)
		}

	}
}
