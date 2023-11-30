package myhandlers

import (
	"github.com/PaulSonOfLars/gotgbot/v2"
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
	err := db.AutoMigrate(&User{}, &GroupInfo{}, &UserFlags{}, &prprCache{}, &YtDlDb{})
	if err != nil {
		panic(err)
	}
}
