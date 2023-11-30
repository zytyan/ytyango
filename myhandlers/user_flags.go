package myhandlers

import (
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/go-co-op/gocron"
	jsoniter "github.com/json-iterator/go"
	"regexp"
	"time"
)

type UserFlags struct {
	ID       int64  `json:"id" gorm:"primaryKey"`
	GroupId  int64  `json:"group_id" gorm:"index"`
	FlagType int    `json:"flag_type"`
	Flag     string `json:"flag"` // json类型
}

const TypeCronMessageFlag = 1

// CronMessageFlag 定时发送 Text 消息
type CronMessageFlag struct {
	UserId int64  `json:"user_id"`
	Cron   string `json:"cron"`
	Text   string `json:"text"`
}

func (u *UserFlags) GetFlag() any {
	switch u.FlagType {
	case TypeCronMessageFlag:
		var flag CronMessageFlag
		err := jsoniter.UnmarshalFromString(u.Flag, &flag)
		if err != nil {
			log.Errorf("unmarshal user flag error %s", err)
			return nil
		}
		return u.Flag
	default:
		log.Errorf("unknown flag type %d", u.FlagType)
		return nil
	}
}

var scheduler = func() *gocron.Scheduler {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		panic(err)
	}
	return gocron.NewScheduler(location)
}()

// 默认每天21点发送
const defaultCron = "0 21 * * *"

func NewCronMessageFlag(groupId, userId int64, text, cron string) *CronMessageFlag {
	_, err := scheduler.Cron(cron).Do(func() {
		_, err := GetMainBot().SendMessage(groupId, text, nil)
		if err != nil {
			log.Warnf("send cron message error %s", err)
			return
		}
	})
	if err != nil {
		log.Errorf("add cron job error %s", err)
		return nil
	}
	return &CronMessageFlag{
		UserId: userId,
		Text:   text,
		Cron:   cron,
	}
}

func NewCronFromMsg(_ *gotgbot.Bot, ctx *ext.Context) error {
	text := getText(ctx)
	NewCronMessageFlag(ctx.EffectiveChat.Id, ctx.EffectiveUser.Id, text, defaultCron)
	return nil
}

var reFlag = regexp.MustCompile(`#flag\b`)

func IsFlagMessage(msg *gotgbot.Message) bool {
	if msg.ForwardFrom != nil {
		return false
	}
	if !GetGroupInfo(msg.Chat.Id).ParseFlags {
		return false
	}
	return reFlag.MatchString(msg.Text)
}
