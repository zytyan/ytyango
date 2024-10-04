package myhandlers

import (
	"fmt"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"main/globalcfg"
	"net/url"
	"path/filepath"
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
	err := db.AutoMigrate(&User{}, &GroupInfo{}, &UserFlags{}, &prprCache{}, &YtDlDb{}, CharacterAttr{})
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
