package genbot

import (
	"context"
	g "main/globalcfg"
	"main/globalcfg/h"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
)

func UpdateGeminiSysPrompt(bot *gotgbot.Bot, ctx *ext.Context) error {
	delete(sysPromptReplacerCache, newTopic(ctx.EffectiveMessage))
	msg := ctx.EffectiveMessage
	text := msg.GetText()
	prompt := h.TrimCmd(text)
	if prompt == "" {
		if msg.ReplyToMessage == nil || msg.ReplyToMessage.GetText() == "" {
			_, err := msg.Reply(bot, `没有找到任何System prompt，请使用 /sysprompt 提示词或使用该命令回复其他消息设置提示词。
您需要使用 /get_sysprompt 获取当前系统提示词， /reset_sysprompt 恢复默认系统提示词。

你可以通过 %VAR% 使用变量，它会自动替换变量名，可使用的变量如下。
TIME: 当前时间，不包含日期
DATE: 当前日期，不含时间
DATETIME: 当前时间和日期
DATETIME_TZ: 包含时区的时间和日期
CHAT_NAME: 当前聊天的名称
BOT_NAME: Bot的名字
BOT_USERNAME: Bot的username
CHAT_TYPE: 聊天类型(group, private)

例：现在是%DATETIME%，当前聊天为%CHAT_NAME%，请根据需要解答群友的问题。
`, nil)
			return err
		}
	}
	err := g.Q.CreateOrUpdateGeminiSystemPrompt(context.Background(), msg.Chat.Id, msg.MessageThreadId, prompt)
	if err != nil {
		_, err = msg.Reply(bot, "设置系统提示词错误: "+err.Error(), nil)
		return err
	}
	_, err = msg.Reply(bot, "成功设置系统提示词:\n"+prompt, nil)
	return err
}
func ResetGeminiSysPrompt(bot *gotgbot.Bot, ctx *ext.Context) error {
	delete(sysPromptReplacerCache, newTopic(ctx.EffectiveMessage))
	err := g.Q.ResetGeminiSystemPrompt(context.Background(), ctx.EffectiveChat.Id, ctx.EffectiveMessage.MessageThreadId)
	if err != nil {
		_, err = ctx.EffectiveMessage.Reply(bot, err.Error(), nil)
		return err
	}
	_, err = ctx.EffectiveMessage.Reply(bot, "已恢复默认提示词", nil)
	return err
}
func GetGeminiSysPrompt(bot *gotgbot.Bot, ctx *ext.Context) error {
	prompt, err := g.Q.GetGeminiSystemPrompt(context.Background(), ctx.EffectiveChat.Id, ctx.EffectiveMessage.MessageThreadId)
	if err != nil {
		_, err = ctx.EffectiveMessage.Reply(bot, gDefaultSysPrompt, nil)
		return err
	}
	_, err = ctx.EffectiveMessage.Reply(bot, prompt, nil)
	return err
}
