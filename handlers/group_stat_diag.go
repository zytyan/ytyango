package handlers

import (
	"context"
	"fmt"
	g "main/globalcfg"
	"main/globalcfg/h"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	jsoniter "github.com/json-iterator/go"
)

func GetRank(bot *gotgbot.Bot, ctx *ext.Context) error {
	stat := g.Q.ChatStatAt(ctx.EffectiveChat.Id, time.Now().Unix())
	if stat == nil || len(stat.UserMsgStat) == 0 {
		_, err := bot.SendMessage(ctx.EffectiveChat.Id, "没有数据", nil)
		return err
	}
	type userCount struct {
		user  int64
		count int64
	}
	tmp := make([]userCount, 0, len(stat.UserMsgStat))
	for u, c := range stat.UserMsgStat {
		if c == nil {
			continue
		}
		tmp = append(tmp, userCount{u, c.MsgCount})
	}
	sort.Slice(tmp, func(i, j int) bool {
		return tmp[i].count > tmp[j].count
	})
	res := make([]string, 0, len(tmp))
	for i, v := range tmp {
		if i >= 10 {
			break
		}
		user, err := g.Q.GetUserById(context.Background(), v.user)
		if err != nil {
			res = append(res, "不知道是谁")
			continue
		}
		res = append(res, fmt.Sprintf("%s: %d", user.Name(), v.count))
	}
	if len(tmp) > 10 {
		sum := int64(0)
		for i := 10; i < len(tmp); i++ {
			sum += tmp[i].count
		}
		res = append(res, fmt.Sprintf("其他人: %d", sum))
	}
	text := strings.Join(res, "\n")
	if text == "" {
		text = "没有数据"
	}
	_, err := bot.SendMessage(ctx.EffectiveChat.Id, text, nil)
	return err
}

func GetCntByTime(bot *gotgbot.Bot, ctx *ext.Context) error {
	stat := g.Q.ChatStatAt(ctx.EffectiveChat.Id, time.Now().Unix())
	if stat == nil {
		_, err := bot.SendMessage(ctx.EffectiveChat.Id, "没有数据", nil)
		return err
	}
	textB, err := jsoniter.Marshal(stat.MsgCountByTime)
	if err != nil {
		_, err = bot.SendMessage(ctx.EffectiveChat.Id, err.Error(), nil)
		return err
	}
	_, err = bot.SendMessage(ctx.EffectiveChat.Id, string(textB), nil)
	return err
}

func SendGroupStat(bot *gotgbot.Bot, ctx *ext.Context) error {
	chatId := ctx.EffectiveChat.Id
	if err := sendChatStat(bot, chatId, time.Now().Add(-24*time.Hour)); err != nil {
		return err
	}
	return sendChatStat(bot, chatId, time.Now())
}

func ForceNewDay(bot *gotgbot.Bot, ctx *ext.Context) error {
	const daySeconds = 24 * 60 * 60
	chatId := ctx.EffectiveChat.Id
	if stat := g.Q.ChatStatAt(chatId, time.Now().Unix()+daySeconds); stat != nil {
		_ = stat.Save(context.Background(), g.Q)
	}
	_, err := bot.SendMessage(ctx.EffectiveChat.Id, "强制新的一天~", nil)
	return err
}

func GroupStatDiagnostic(bot *gotgbot.Bot, ctx *ext.Context) error {
	filename := fmt.Sprintf("groupstat_%d.txt", ctx.EffectiveChat.Id)
	f, err := os.Create(filename)
	if err != nil {
		_, err = bot.SendMessage(ctx.EffectiveChat.Id, "创建文件错误", nil)
		return err
	}
	defer f.Close()
	defer os.Remove(filename)

	if statJob != nil {
		_, _ = f.WriteString(fmt.Sprintf("run count: %d, next run: %s\n", statJob.RunCount(), statJob.NextRun().Format("2006-01-02 15:04:05")))
	} else {
		_, _ = f.WriteString("stat scheduler not started\n")
	}
	cfg := g.GetConfig()
	for _, chatId := range cfg.MyChats {
		_, _ = f.WriteString(fmt.Sprintf("chat %d\n", chatId))
		nowStat, tz, err := g.Q.ChatStatOfDay(context.Background(), chatId, time.Now().Unix())
		if err == nil {
			_, _ = f.WriteString(fmt.Sprintf("today msg count: %d (tz=%d)\n", nowStat.MessageCount, tz))
		}
		yesterdayStat, _, err := g.Q.ChatStatOfDay(context.Background(), chatId, time.Now().Add(-24*time.Hour).Unix())
		if err == nil {
			_, _ = f.WriteString(fmt.Sprintf("yesterday msg count: %d\n", yesterdayStat.MessageCount))
		}
	}

	_, err = bot.SendDocument(ctx.EffectiveChat.Id, h.LocalFile(filename), nil)
	return err
}
