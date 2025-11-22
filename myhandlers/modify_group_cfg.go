package myhandlers

import (
	"cmp"
	"context"
	"fmt"
	g "main/globalcfg"
	"main/globalcfg/q"
	"reflect"
	"slices"
	"strings"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
)

const GroupConfigModifyPrefix = "gcfg:"

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

type btnField struct {
	Name     string
	Text     string
	Value    bool
	Position [2]int
}

func getBtnTxtFields(cfg *q.ChatCfg) []btnField {
	v := reflect.ValueOf(cfg).Elem()
	t := v.Type()

	var fields []btnField
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("btnTxt")
		posTag := field.Tag.Get("pos")
		if tag == "" || posTag == "" || field.Type.Kind() != reflect.Bool {
			continue
		}
		var row, col int
		_, _ = fmt.Sscanf(posTag, "%d,%d", &row, &col)
		fields = append(fields, btnField{
			Name:     field.Name,
			Text:     tag,
			Value:    v.Field(i).Bool(),
			Position: [2]int{row, col},
		})
	}
	return fields
}

func getBtnTxtFieldByName(cfg *q.ChatCfg, name string) btnField {
	v := reflect.ValueOf(cfg).Elem()
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("btnTxt")
		posTag := field.Tag.Get("pos")
		if tag == "" || posTag == "" || field.Type.Kind() != reflect.Bool {
			continue
		}
		if field.Name == name {
			var row, col int
			_, _ = fmt.Sscanf(posTag, "%d,%d", &row, &col)
			return btnField{
				Name:     field.Name,
				Text:     tag,
				Value:    v.Field(i).Bool(),
				Position: [2]int{row, col},
			}
		}
	}
	return btnField{Name: "???"}
}

func setFieldByName(cfg *q.ChatCfg, field string, val bool) error {
	v := reflect.ValueOf(cfg).Elem()
	t := v.Type()

	f, ok := t.FieldByName(field)
	if !ok {
		return fmt.Errorf("no such field: %s", field)
	}
	if f.Tag.Get("btnTxt") == "" {
		return fmt.Errorf("field %s has no btnTxt tag (not allowed to modify)", field)
	}
	fieldVal := v.FieldByName(field)
	if !fieldVal.CanSet() || fieldVal.Kind() != reflect.Bool {
		return fmt.Errorf("cannot set field: %s", field)
	}
	fieldVal.SetBool(val)
	return nil
}

func generateGroupModifyReplyMarkup(cfg *q.ChatCfg) gotgbot.InlineKeyboardMarkup {
	fields := getBtnTxtFields(cfg)
	if len(fields) == 0 {
		return gotgbot.InlineKeyboardMarkup{}
	}
	mf := slices.MaxFunc(fields, func(a, b btnField) int {
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
	for _, field := range fields {
		x, y := field.Position[0]-1, field.Position[1]-1
		btns[x][y] = gotgbot.InlineKeyboardButton{
			Text:         fmt.Sprintf("%s %s", boolToEmoji(field.Value), field.Text),
			CallbackData: GroupConfigModifyPrefix + field.Name + ":" + boolToTF(field.Value),
		}
	}
	return gotgbot.InlineKeyboardMarkup{InlineKeyboard: btns}
}

func ModifyGroupConfigByButton(bot *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	if msg == nil {
		return fmt.Errorf("ModifyGroupConfigByButton: message should not be nil")
	}
	cfg, err := g.Q.GetChatById(context.Background(), msg.Chat.Id)
	if err != nil {
		return err
	}
	cmdList := strings.Split(ctx.CallbackQuery.Data, ":")
	if len(cmdList) < 3 {
		return fmt.Errorf("ModifyGroupConfigByButton: invalid command")
	}
	field, arg := cmdList[1], cmdList[2]
	argBool := arg == "T"
	if err = setFieldByName(cfg, field, !argBool); err != nil {
		_, _ = ctx.CallbackQuery.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{
			Text:      err.Error(),
			ShowAlert: true,
			CacheTime: 0,
		})
		return err
	}
	if err = cfg.Save(g.Q); err != nil {
		log.Warnf("update chat cfg failed: %v", err)
	}
	_, _, err = ctx.EffectiveMessage.EditReplyMarkup(bot, &gotgbot.EditMessageReplyMarkupOpts{
		ReplyMarkup: generateGroupModifyReplyMarkup(cfg),
	})
	showText := getBtnTxtFieldByName(cfg, field).Text
	_, _ = ctx.CallbackQuery.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{
		Text:      fmt.Sprintf("%s %s -> %s", showText, boolToEmoji(argBool), boolToEmoji(!argBool)),
		CacheTime: 0,
	})

	return err
}

func ShowChatCfg(bot *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	if msg == nil {
		return fmt.Errorf("ShowChatCfg: message should not be nil")
	}
	cfg, err := g.Q.GetChatById(context.Background(), msg.Chat.Id)
	if err != nil {
		return err
	}
	_, err = bot.SendMessage(msg.Chat.Id, "当前群组的信息如下", &gotgbot.SendMessageOpts{
		ReplyMarkup: generateGroupModifyReplyMarkup(cfg),
	})
	return err
}
