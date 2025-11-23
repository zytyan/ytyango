package myhandlers

import (
	"fmt"
	"html"
	"math/rand"
	"net/url"
	"strconv"
	"time"
	"unicode"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
)

func splitCommand(command string) (string, string) {
	i := 0
	cmd, args := "", ""
	for j, r := range command {
		if unicode.IsSpace(r) {
			if i == 0 {
				cmd = command[:j]
			}
			i = j
		} else {
			if i != 0 {
				args = command[j:]
				break
			}
		}
	}
	if i == 0 {
		return command, ""
	}
	return cmd, args
}

func getText(ctx *ext.Context) string {
	return getTextMsg(ctx.EffectiveMessage)
}

func getTextMsg(msg *gotgbot.Message) string {
	if msg == nil {
		return ""
	}
	if msg.Text != "" {
		return msg.Text
	}
	return msg.Caption
}

func parseIntDefault(str string, defaultValue int) int {
	if i, err := strconv.Atoi(str); err == nil {
		return i
	}
	return defaultValue
}

func Roll(bot *gotgbot.Bot, ctx *ext.Context) error {
	start, end := 1, 20
	args := ctx.Args()
	switch len(args) {
	case 1:
		break
	case 2:
		end = parseIntDefault(args[1], end)
	case 3:
		fallthrough
	default:
		start, end = parseIntDefault(args[1], start), parseIntDefault(args[2], end)
		if start > end {
			start, end = end, start
		}
	}
	rd := rand.Intn(end-start+1) + start
	reply := fmt.Sprintf("在%d-%d的roll点中，你掷出了%d\n本消息将在五分钟后删除", start, end, rd)
	message, err := ctx.Message.Reply(bot, reply, nil)
	if err != nil {
		log.Warnf("reply failed: %s", err)
		return err
	}
	time.AfterFunc(5*time.Second, func() {
		_, err := message.Delete(bot, nil)
		if err != nil {
			log.Warnf("delete bot message failed: %s", err)
			return
		}
		_, _ = ctx.Message.Delete(bot, nil)
	})
	return nil
}

func Google(bot *gotgbot.Bot, ctx *ext.Context) error {
	text := getText(ctx)
	_, query := splitCommand(text)
	if query == "" {
		_, err := ctx.Message.Reply(bot, "好消息，本群已和Google达成战略合作，以后有问题可以去Google搜索，不用来群里问啦！\n"+
			"此外，您还可以使用 /google 关键字 来生成搜索链接。", nil)
		return err
	}
	googleUrl := "https://www.google.com/search?q=" + url.QueryEscape(query)
	_, err := ctx.Message.Reply(bot, fmt.Sprintf(`Google: <a href="%s">%s</a>`, googleUrl,
		html.EscapeString(query)), &gotgbot.SendMessageOpts{ParseMode: "HTML"})
	return err
}
func Wiki(bot *gotgbot.Bot, ctx *ext.Context) error {
	text := getText(ctx)
	_, query := splitCommand(text)
	if query == "" {
		_, err := ctx.Message.Reply(bot, `<a href="https://zh.wikipedia.org/wiki/">维基百科</a>`, &gotgbot.SendMessageOpts{ParseMode: "HTML"})
		if err != nil {
			log.Warnf("wiki reply failed: %s", err)
		}
		return err
	}
	wikiUrl := "https://zh.wikipedia.org/w/index.php?search=" + url.QueryEscape(query)
	htmlEscaped := html.EscapeString(query)
	_, err := ctx.Message.Reply(bot, fmt.Sprintf(
		`Wiki: <a href="%s">%s的wiki搜索结果</a>%s结果不对？尝试<a href="%s">在Google搜索维基百科中的%s</a>`,
		wikiUrl,
		htmlEscaped,
		"\n\n",
		"https://www.google.com/search?q="+url.QueryEscape(query)+"+site:wikipedia.org",
		htmlEscaped), &gotgbot.SendMessageOpts{ParseMode: "HTML"})
	if err != nil {
		log.Warnf("wiki reply failed: %s", err)
	}
	return err
}
