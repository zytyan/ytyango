package myhandlers

import (
	"fmt"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"main/helpers/mathparser"
	"regexp"
	"strings"
)

func SolveMath(bot *gotgbot.Bot, ctx *ext.Context) (err error) {
	text := ctx.Message.Text
	force := false
	text = mathReplacer.Replace(text)
	if strings.HasPrefix(text, "/") {
		_, text = splitCommand(text)
		force = true
	}
	res, err := mathparser.ParseString(text)
	if err != nil {
		if force {
			_, _ = ctx.EffectiveMessage.Reply(bot,
				fmt.Sprintf("计算失败, error: %s", err.Error()),
				nil)
		}
		return nil
	}
	_, _ = ctx.EffectiveMessage.Reply(bot,
		fmt.Sprintf("%s = %s", text, res.ToText()), nil)
	return
}

var mathReplacer = func() *strings.Replacer {
	src := "（）【】！￥，。？“”‘’～"
	dst := "()[]!$,.?\"\"''~"
	srcL := make([]rune, 0, len(dst))
	dstL := make([]rune, 0, len(dst))
	for _, r := range src {
		srcL = append(srcL, r)
	}
	for _, r := range dst {
		dstL = append(dstL, r)
	}
	if len(srcL) != len(dstL) {
		panic("len(srcL) != len(dstL)")
	}
	outL := make([]string, 0, len(dst)*2)
	for i, r := range srcL {
		outL = append(outL, string(r), string(dstL[i]))
	}
	replacer := strings.NewReplacer(outL...)
	return replacer
}()

var mathRe = regexp.MustCompile(`^[a-zA-Z0-9+\-*/^.()]+$`)
var pureDigestRe = regexp.MustCompile(`(^[0-9\.]+$)|(0x[0-9a-fA-F]+)|pi|e`)

func NeedSolve(msg *gotgbot.Message) bool {
	if !GetGroupInfo(msg.Chat.Id).AutoCalculate {
		return false
	}
	text := msg.Text
	if strings.HasPrefix(text, "/") || pureDigestRe.MatchString(text) {
		return false
	}
	return mathRe.MatchString(text)
}
