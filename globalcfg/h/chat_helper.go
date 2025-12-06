package h

import (
	"context"
	g "main/globalcfg"
	"os"

	"github.com/PaulSonOfLars/gotgbot/v2"
)

func ChatAutoCvtBili(chatId int64) bool {
	cfg, err := g.Q.ChatCfgById(context.Background(), chatId)
	if err != nil {
		return false
	}
	return cfg.AutoCvtBili
}

func ChatAutoOcr(chatId int64) bool {
	cfg, err := g.Q.ChatCfgById(context.Background(), chatId)
	if err != nil {
		return false
	}
	return cfg.AutoOcr
}

func ChatAutoCalculate(chatId int64) bool {
	cfg, err := g.Q.ChatCfgById(context.Background(), chatId)
	if err != nil {
		return false
	}
	return cfg.AutoCalculate
}

func ChatAutoExchange(chatId int64) bool {
	cfg, err := g.Q.ChatCfgById(context.Background(), chatId)
	if err != nil {
		return false
	}
	return cfg.AutoExchange
}

func ChatAutoCheckAdult(chatId int64) bool {
	cfg, err := g.Q.ChatCfgById(context.Background(), chatId)
	if err != nil {
		return false
	}
	return cfg.AutoCheckAdult
}

func ChatSaveMessages(chatId int64) bool {
	cfg, err := g.Q.ChatCfgById(context.Background(), chatId)
	if err != nil {
		return false
	}
	return cfg.SaveMessages
}

func ChatEnableCoc(chatId int64) bool {
	cfg, err := g.Q.ChatCfgById(context.Background(), chatId)
	if err != nil {
		return false
	}
	return cfg.EnableCoc
}

func ChatRespNsfwMsg(chatId int64) bool {
	cfg, err := g.Q.ChatCfgById(context.Background(), chatId)
	if err != nil {
		return false
	}
	return cfg.RespNsfwMsg
}

func GetFileBytes(bot *gotgbot.Bot, fileId string) ([]byte, error) {
	f, err := bot.GetFile(fileId, nil)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(f.FilePath)
}
