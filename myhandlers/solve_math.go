package myhandlers

import (
	"fmt"
	"main/globalcfg/h"
	"main/helpers/mathparser"
	"math/big"
	"strings"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
)

func ratToText(r *big.Rat) string {
	return strings.TrimRight(strings.TrimRight(r.FloatString(4), "0"), ".")
}

func SolveMath(bot *gotgbot.Bot, ctx *ext.Context) (err error) {
	text := ctx.Message.Text
	force := false
	text = mathReplacer.Replace(text)
	if strings.HasPrefix(text, "/") {
		_, text = splitCommand(text)
		force = true
	}
	res, err := mathparser.Evaluate(text)
	if err != nil {
		if force {
			_, _ = ctx.EffectiveMessage.Reply(bot,
				fmt.Sprintf("计算失败, error: %s", err.Error()),
				nil)
		}
		return nil
	}
	_, _ = ctx.EffectiveMessage.Reply(bot,
		fmt.Sprintf("%s = %s", text, ratToText(res)), nil)
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

func NeedSolve(msg *gotgbot.Message) bool {
	if !h.ChatAutoCalculate(msg.Chat.Id){
		return false
	}
	text := msg.Text
	if strings.HasPrefix(text, "/") {
		return false
	}
	return mathparser.FastCheck(text)
}
