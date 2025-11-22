package h

import (
	"context"
	g "main/globalcfg"
)

func ChatAutoCvtBili(chatId int64) bool {
	cfg, err := g.Q.GetChatById(context.Background(), chatId)
	if err != nil {
		return false
	}
	return cfg.AutoCvtBili
}

func ChatAutoOcr(chatId int64) bool {
	cfg, err := g.Q.GetChatById(context.Background(), chatId)
	if err != nil {
		return false
	}
	return cfg.AutoOcr
}

func ChatAutoCalculate(chatId int64) bool {
	cfg, err := g.Q.GetChatById(context.Background(), chatId)
	if err != nil {
		return false
	}
	return cfg.AutoCalculate
}

func ChatAutoExchange(chatId int64) bool {
	cfg, err := g.Q.GetChatById(context.Background(), chatId)
	if err != nil {
		return false
	}
	return cfg.AutoExchange
}

func ChatAutoCheckAdult(chatId int64) bool {
	cfg, err := g.Q.GetChatById(context.Background(), chatId)
	if err != nil {
		return false
	}
	return cfg.AutoCheckAdult
}

func ChatSaveMessages(chatId int64) bool {
	cfg, err := g.Q.GetChatById(context.Background(), chatId)
	if err != nil {
		return false
	}
	return cfg.SaveMessages
}

func ChatEnableCoc(chatId int64) bool {
	cfg, err := g.Q.GetChatById(context.Background(), chatId)
	if err != nil {
		return false
	}
	return cfg.EnableCoc
}

func ChatRespNsfwMsg(chatId int64) bool {
	cfg, err := g.Q.GetChatById(context.Background(), chatId)
	if err != nil {
		return false
	}
	return cfg.RespNsfwMsg
}
