package myhandlers

import (
	"cmp"
	"fmt"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"slices"
	"strings"
)

const GroupConfigModifyPrefix = "gcfg:"

// gcfg:FieldName:T/F

func boolToEmoji(b bool) string {
	if b {
		return "✅"
	}
	return "❌"
}

func boolToTF(b bool) string {
	if b {
		return "T"
	}
	return "F"
}

func generateGroupModifyReplyMarkup(groupInfo *GroupInfo) gotgbot.InlineKeyboardMarkup {
	// 注意: Positon 为从 1 开始的索引，使用时需要 -1
	fields := groupInfo.GetBtnTxtFields()
	mf := slices.MaxFunc(fields, func(a, b BtnField) int {
		return cmp.Compare(a.Position[0], b.Position[0])
	})
	rows := make([]int, mf.Position[0])
	for _, field := range fields {
		if field.Position[1] > rows[field.Position[0]-1] {
			rows[field.Position[0]-1] = field.Position[1]
		}
	}
	btns := make([][]gotgbot.InlineKeyboardButton, mf.Position[0])
	for i, row := range rows {
		btns[i] = make([]gotgbot.InlineKeyboardButton, row)
	}
	markup := gotgbot.InlineKeyboardMarkup{InlineKeyboard: btns}
	for _, field := range fields {
		x, y := field.Position[0]-1, field.Position[1]-1
		if btns[x][y].Text != "" {
			log.Errorf("群组button存在位置重复的情况")
		}
		btns[x][y] = gotgbot.InlineKeyboardButton{
			Text:         fmt.Sprintf("%s %s", boolToEmoji(field.Value), field.Text),
			CallbackData: GroupConfigModifyPrefix + field.Name + ":" + boolToTF(field.Value),
		}
	}
	return markup
}

func ModifyGroupConfigByButton(bot *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	if msg == nil {
		return fmt.Errorf("ModifyGroupConfigByButton: message should not be nil")
	}
	groupInfo, err := getGroupInfo(msg.Chat.Id)
	if err != nil {
		return err
	}
	groupInfo.mu.Lock()
	defer groupInfo.mu.Unlock()
	cmdList := strings.Split(ctx.CallbackQuery.Data, ":")
	if len(cmdList) < 3 {
		return fmt.Errorf("ModifyGroupConfigByButton: invalid command")
	}
	field, arg := cmdList[1], cmdList[2]
	argBool := false
	if arg == "T" {
		argBool = true
	}
	err = groupInfo.SetFieldByName(field, !argBool)
	if err != nil {
		_, err = ctx.CallbackQuery.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{
			Text:      err.Error(),
			ShowAlert: true,
			CacheTime: 0,
		})
	}
	_, _, err = ctx.EffectiveMessage.EditReplyMarkup(bot, &gotgbot.EditMessageReplyMarkupOpts{
		ReplyMarkup: generateGroupModifyReplyMarkup(groupInfo),
	})
	showText := groupInfo.GetBtnTxtFieldByName(cmdList[1]).Text
	groupInfo.Update()
	_, err = ctx.CallbackQuery.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{
		Text:      fmt.Sprintf("%s %s -> %s", showText, boolToEmoji(argBool), boolToEmoji(!argBool)),
		CacheTime: 0,
	})

	return err
}

func ShowChatCfg(bot *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	if msg == nil {
		return fmt.Errorf("ModifyGroupConfigByButton: message should not be nil")
	}
	groupInfo, err := getGroupInfo(msg.Chat.Id)
	if err != nil {
		return err
	}
	_, err = bot.SendMessage(msg.Chat.Id, "当前群组的信息如下", &gotgbot.SendMessageOpts{
		ReplyMarkup: generateGroupModifyReplyMarkup(groupInfo),
	})
	return err
}
