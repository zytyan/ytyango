package myhandlers

import (
	"errors"
	"fmt"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"main/helpers/exchange"
)

func IsCalcExchangeRate(msg *gotgbot.Message) bool {
	if !GetGroupInfo(msg.Chat.Id).AutoExchange {
		return false
	}
	return exchange.IsExchangeRateCalc(getTextMsg(msg))
}

func ToggleAutoExchange(bot *gotgbot.Bot, ctx *ext.Context) error {
	group := GetGroupInfo(ctx.EffectiveChat.Id)
	if group == nil {
		return nil
	}
	groupInfoWMutex.Lock()
	defer groupInfoWMutex.Unlock()
	group.AutoExchange = !group.AutoExchange
	text := fmt.Sprintf("群组自动汇率换算切换为%t", group.AutoExchange)
	_, err := ctx.EffectiveMessage.Reply(bot, text, nil)
	return err
}

var exchangeAlias = map[string]string{
	"RMB": "CNY",
	"NTD": "TWD",
}

func ExchangeRateCalc(bot *gotgbot.Bot, ctx *ext.Context) error {
	req, err := exchange.ParseExchangeRate(getText(ctx))
	if err != nil {
		return err
	}
	rate, err := exchange.GetExchangeRateWithAlias(req, exchangeAlias)
	if err != nil {
		if errors.Is(err, exchange.NotAValidExchangeReq) {
			return nil
		} else if errors.Is(err, exchange.ErrFromNotFound) {
			return nil
		} else if errors.Is(err, exchange.ErrToNotFound) {
			return nil
		} else if errors.Is(err, exchange.CashNotAvail) {
			return nil
		}
		_, err = ctx.EffectiveMessage.Reply(bot, err.Error(), nil)
		return err
	}
	updateAt := rate.UpdateAt.Format("2006-01-02 15:04:05")
	text := fmt.Sprintf("%.4f %s = %.4f %s\n汇率更新于: %s",
		req.Amount, req.From, rate.Result, req.To, updateAt)
	_, err = ctx.EffectiveMessage.Reply(bot, text, nil)
	return err
}
