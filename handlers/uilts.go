package handlers

import (
	"sync"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
)

var mainBot *gotgbot.Bot

func SetMainBot(bot *gotgbot.Bot) {
	mainBot = bot
}

func GetMainBot() *gotgbot.Bot {
	return mainBot
}

func MakeReplyToMsgID(msgId int64) *gotgbot.ReplyParameters {
	if msgId == 0 {
		return nil
	}
	return &gotgbot.ReplyParameters{
		MessageId: msgId,
	}
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
			log.Warnf("answer %s[%s] callback ran twice", ctx.CallbackQuery.Id, ctx.CallbackQuery.Data)
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

func getChatName(chat *gotgbot.Chat) string {
	if chat.Title != "" {
		return chat.Title
	}
	if chat.LastName == "" {
		return chat.FirstName
	}
	return chat.FirstName + " " + chat.LastName
}
