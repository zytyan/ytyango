package myhandlers

import (
	"fmt"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/google/uuid"
	"golang.org/x/text/width"
	"main/globalcfg"
	"main/helpers/cocdice"
	"regexp"
	"strconv"
	"strings"
)

var dndDiceRe = regexp.MustCompile(`^((\d+)\s*)?([dDbBpP])\s*(\d+)?(\s*([+-])(\d+))?\s*(/?\s*([\w\p{Han}\p{Hiragana}\p{Katakana}]+))?$`)
var setAttrRe = regexp.MustCompile(`^([a-zA-Z\p{Han}\p{Hiragana}\p{Katakana}][a-zA-Z0-9\p{Han}\p{Hiragana}\p{Katakana}]{0,20})\s*(\+=?|-=?|=)\s*(\d+)$`)

type CharacterAttr struct {
	ID        int `gorm:"primaryKey"`
	UserID    int64
	AttrName  string
	AttrValue string
}

func defaultAtoi(s string, d int) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		return d
	}
	return i
}

func IsDndDice(msg *gotgbot.Message) bool {
	if msg.Chat.Type != "private" && !GetGroupInfo(msg.Chat.Id).CoCEnabled {
		return false
	}
	text := width.Narrow.String(msg.Text)
	return dndDiceRe.MatchString(text)
}

func IsSetDndAttr(msg *gotgbot.Message) bool {
	if msg.Chat.Type != "private" && !GetGroupInfo(msg.Chat.Id).CoCEnabled {
		return false
	}
	text := width.Narrow.String(msg.Text)
	text = strings.SplitN(text, "\n", 2)[0]
	return setAttrRe.MatchString(text)
}
func getUserName(user *gotgbot.User) string {
	if user == nil {
		return "未知"
	}
	if user.LastName == "" {
		return user.FirstName
	}
	return user.FirstName + " " + user.LastName
}
func getAttrTarget(ctx *ext.Context) (int64, string) {
	if ctx.EffectiveMessage.ReplyToMessage != nil && ctx.EffectiveMessage.ReplyToMessage.From != nil {
		from := ctx.EffectiveMessage.ReplyToMessage.From
		return from.Id, getUserName(from)
	}
	return ctx.EffectiveSender.Id(), ctx.EffectiveSender.Name()
}
func SetDndAttr(bot *gotgbot.Bot, ctx *ext.Context) (err error) {
	text := width.Narrow.String(ctx.EffectiveMessage.Text)
	lines := strings.Split(text, "\n")
	buf := strings.Builder{}
	userId, name := getAttrTarget(ctx)
	buf.WriteString(fmt.Sprintf("用户：%s\n", name))
	modified := false
	for _, line := range lines {
		matches := setAttrRe.FindStringSubmatch(line)
		attr := CharacterAttr{UserID: userId, AttrName: matches[1]}
		err = globalcfg.GetDb().Model(&CharacterAttr{}).Where(&attr).First(&attr).Error
		oldValue := attr.AttrValue
		if oldValue == "" {
			oldValue = "empty"
		}
		switch matches[2] {
		case "+":
			if err != nil {
				continue
			}
			fallthrough
		case "+=":
			attr.AttrValue = strconv.Itoa(defaultAtoi(attr.AttrValue, 0) + defaultAtoi(matches[3], 0))
		case "-":
			if err != nil {
				continue
			}
			fallthrough
		case "-=":
			attr.AttrValue = strconv.Itoa(defaultAtoi(attr.AttrValue, 0) - defaultAtoi(matches[3], 0))
		default:
			attr.AttrValue = matches[3]
		}
		modified = true
		buf.WriteString(fmt.Sprintf("%s : %s -> %s\n", attr.AttrName, oldValue, attr.AttrValue))
		globalcfg.GetDb().Save(&attr)
	}
	if !modified {
		return nil
	}
	s := buf.String()
	s = s[:len(s)-1]
	_, err = ctx.EffectiveMessage.Reply(bot, "设置成功\n"+s, nil)
	return err
}

func getAbility(m string, user int64) (int, error) {
	ability, err := strconv.Atoi(m)
	if err != nil {
		attr := CharacterAttr{UserID: user, AttrName: m}
		err = globalcfg.GetDb().Model(&CharacterAttr{}).Where(&attr).First(&attr).Error
		if err != nil {
			return 0, err
		}
		return strconv.Atoi(attr.AttrValue)

	}
	return ability, nil
}

func ListDndAttr(bot *gotgbot.Bot, ctx *ext.Context) (err error) {
	var attrs []CharacterAttr
	text := width.Narrow.String(ctx.EffectiveMessage.Text)
	texts := strings.SplitN(text, " ", 2)
	var re *regexp.Regexp
	if len(texts) > 1 {
		text = texts[1]
		text = strings.TrimSpace(text)
		re, _ = regexp.Compile(text)
	}
	log.Infof("text: %s", text)
	userId, name := getAttrTarget(ctx)
	globalcfg.GetDb().Model(&CharacterAttr{}).Where("user_id = ?", userId).Find(&attrs)
	var result strings.Builder
	result.WriteString(fmt.Sprintf("用户：%s\n", name))
	for _, attr := range attrs {
		if re == nil || re.FindString(attr.AttrName) != "" {
			result.WriteString(attr.AttrName)
			result.WriteString(" = ")
			result.WriteString(attr.AttrValue)
			result.WriteString("\n")
		}
	}
	if result.Len() == 0 {
		result.WriteString(fmt.Sprintf("没有找到属性：%s", text))
	}
	_, err = ctx.EffectiveMessage.Reply(bot, result.String(), nil)
	return err
}

func DelDndAttr(bot *gotgbot.Bot, ctx *ext.Context) (err error) {
	text := width.Narrow.String(ctx.EffectiveMessage.Text)
	lines := strings.Split(text, "\n")
	for _, line := range lines[1:] {
		attr := CharacterAttr{UserID: ctx.EffectiveSender.Id(), AttrName: line}
		globalcfg.GetDb().Model(&CharacterAttr{}).Where(&attr).Delete(&attr)
	}
	_, err = ctx.EffectiveMessage.Reply(bot, "删除成功", nil)
	return err
}

func DndDice(bot *gotgbot.Bot, ctx *ext.Context) (err error) {
	text := width.Narrow.String(ctx.EffectiveMessage.Text)
	matches := dndDiceRe.FindStringSubmatch(text)
	diceCount := defaultAtoi(matches[2], 1)
	diceType := strings.ToLower(matches[3])
	diceFace := defaultAtoi(matches[4], 100)
	modifier, _ := strconv.Atoi(matches[7])
	var ability int
	if matches[9] != "" {
		ability, err = getAbility(matches[9], ctx.EffectiveSender.Id())
		if err != nil {
			log.Infof("get ability error: %s", err)
			return nil
		}
	}
	diceCommand := cocdice.DiceCommand{Arg1: diceCount, Arg2: diceFace, Modifier: modifier, Ability: ability}
	switch diceType {
	case "d":
		diceCommand.Type = cocdice.NormalDice
	case "b":
		diceCommand.Type = cocdice.BonusDice
	case "p":
		diceCommand.Type = cocdice.PenaltyDice
	}
	result := diceCommand.Roll()
	_, err = ctx.EffectiveMessage.Reply(bot, result, &gotgbot.SendMessageOpts{ParseMode: "HTML"})
	return err
}

func CoCHelp(bot *gotgbot.Bot, ctx *ext.Context) (err error) {
	helpText := `<b>在使用CoC辅助之前，请先使用 /toggle_coc 命令启用CoC功能</b>

<b>骰子功能</b>
<b>用法：</b>
1. [骰子数量]d[骰子面数] [+/-调整值]
2. [骰子数量]b[骰子面数] [+/-调整值]
3. [骰子数量]p[骰子面数] [+/-调整值]
<b>示例：</b>
1. 3d6 投掷3个6面骰子
2. 2b100 投掷2个100面奖励骰
3. 1p100 投掷1个100面惩罚骰
4. 1d20+5 投掷1个20面骰子并加5
`
	_, err = ctx.EffectiveMessage.Reply(bot, helpText, &gotgbot.SendMessageOpts{ParseMode: "HTML"})
	return err
}

var battleMap = make(map[string]*cocdice.BattleRound)

func buildBattleKeyboard(uid string) gotgbot.InlineKeyboardMarkup {
	return gotgbot.InlineKeyboardMarkup{
		InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
			{gotgbot.InlineKeyboardButton{CallbackData: "battle:next:" + uid, Text: "下一回合"}},
			{gotgbot.InlineKeyboardButton{CallbackData: "battle:stop:" + uid, Text: "结束战斗"}},
		},
	}
}

var battleGroupIdToUuid = make(map[int64]string)

func NewBattle(bot *gotgbot.Bot, ctx *ext.Context) (err error) {
	gid := ctx.EffectiveChat.Id
	log.Infof("battle gid: %d", gid)
	if _, ok := battleGroupIdToUuid[gid]; ok {
		_, err = ctx.EffectiveMessage.Reply(bot, "已经有一个战斗在进行中", nil)
		return err
	}
	uid, _ := uuid.NewUUID()
	uidStr := uid.String()
	battle := cocdice.NewFromText(ctx.EffectiveMessage.Text)
	battleGroupIdToUuid[gid] = uidStr
	battleMap[uidStr] = battle
	_, err = ctx.EffectiveMessage.Reply(bot, battle.String(), &gotgbot.SendMessageOpts{ParseMode: "HTML",
		ReplyMarkup: buildBattleKeyboard(uidStr),
	})
	return err
}

func IsNextRound(msg *gotgbot.CallbackQuery) bool {
	return strings.HasPrefix(msg.Data, "battle:next:")
}

func IsStopBattle(msg *gotgbot.CallbackQuery) bool {
	return strings.HasPrefix(msg.Data, "battle:stop:")
}

func StopBattle(bot *gotgbot.Bot, ctx *ext.Context) (err error) {
	uid := getBattleUid(ctx.CallbackQuery.Data)
	delete(battleMap, uid)
	delete(battleGroupIdToUuid, ctx.EffectiveChat.Id)
	_, _ = ctx.CallbackQuery.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{Text: "战斗结束", ShowAlert: true})
	_, err = ctx.EffectiveMessage.Reply(bot, "战斗结束", nil)
	return err
}

func NextRound(bot *gotgbot.Bot, ctx *ext.Context) (err error) {
	uid := getBattleUid(ctx.CallbackQuery.Data)
	battle, ok := battleMap[uid]
	if !ok {
		_, err = ctx.EffectiveMessage.Reply(bot, "战斗已经结束", nil)
		return err
	}
	battle.NextCharacter()
	_, _ = ctx.CallbackQuery.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{Text: "下一回合", ShowAlert: false})
	_, err = ctx.EffectiveMessage.Reply(bot, battle.String(), &gotgbot.SendMessageOpts{ParseMode: "HTML",
		ReplyMarkup: buildBattleKeyboard(uid),
	})
	return err
}

var reBattleCmd = regexp.MustCompile(`^(add|chg|del|stat)`)

func getBattleUid(data string) string {
	return strings.Split(data, ":")[2]
}

func IsBattleCommand(msg *gotgbot.Message) bool {
	gid := msg.Chat.Id
	if _, ok := battleGroupIdToUuid[gid]; ok {
		return reBattleCmd.MatchString(msg.Text)
	}
	if msg.ReplyToMessage == nil {
		return false
	}
	replyToMsg := msg.ReplyToMessage
	if replyToMsg.ReplyMarkup == nil || len(replyToMsg.ReplyMarkup.InlineKeyboard) == 0 || len(replyToMsg.ReplyMarkup.InlineKeyboard[0]) == 0 {
		return false
	}
	if !strings.HasPrefix(replyToMsg.ReplyMarkup.InlineKeyboard[0][0].CallbackData, "battle:") {
		return false
	}
	log.Infof("battle command: %s", msg.Text)
	return reBattleCmd.MatchString(msg.Text)
}

func ExecuteBattleCommand(bot *gotgbot.Bot, ctx *ext.Context) (err error) {
	uid := battleGroupIdToUuid[ctx.EffectiveChat.Id]
	battle, ok := battleMap[uid]
	if !ok {
		_, err = ctx.EffectiveMessage.Reply(bot, "战斗已经结束", nil)
		return err
	}
	textList := strings.Split(ctx.EffectiveMessage.Text, "\n")
	errList := make([]string, 0)
	for _, text := range textList {
		log.Infof("battle command: %s", text)
		err = battle.ParseCommand(text)
		if err != nil {
			errList = append(errList, err.Error())
		}
	}
	if len(errList) > 0 {
		errStr := strings.Join(errList, "\n")
		_, err = ctx.EffectiveMessage.Reply(bot, errStr, nil)
		return err
	}
	_, err = ctx.EffectiveMessage.Reply(bot, battle.String(), &gotgbot.SendMessageOpts{ParseMode: "HTML",
		ReplyMarkup: buildBattleKeyboard(uid),
	})
	return err
}

func ToggleCoC(bot *gotgbot.Bot, ctx *ext.Context) (err error) {
	group := GetGroupInfo(ctx.EffectiveChat.Id)
	group.CoCEnabled = !group.CoCEnabled
	globalcfg.GetDb().Save(&group)
	if group.CoCEnabled {
		_, err = ctx.EffectiveMessage.Reply(bot, "CoC功能已启用", nil)
	} else {
		_, err = ctx.EffectiveMessage.Reply(bot, "CoC功能已禁用", nil)
	}
	return err
}
