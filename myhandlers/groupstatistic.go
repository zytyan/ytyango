package myhandlers

import (
	"main/groupstatv2"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/go-co-op/gocron"
)

func AddNewMsg(_ *gotgbot.Bot, ctx *ext.Context) error {
	groupstatv2.AddMsg(ctx.EffectiveMessage)
	return nil
}

var sendScheduler *gocron.Scheduler
var job *gocron.Job

func init() {
	err := groupstatv2.LoadFromFile()
	if err != nil {
		log.Errorf("Load groupstat from file error: %s", err)
	}
	sendScheduler = gocron.NewScheduler(time.Local)
	job, err = sendScheduler.Every(1).Day().At("08:00").Do(func() {
		log.Info("send group stat")
	})
	sendScheduler.StartAsync()
	if err != nil {
		panic(err)
	}
}
