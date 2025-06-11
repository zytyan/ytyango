package myhandlers

import (
	"fmt"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	jsoniter "github.com/json-iterator/go"

	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

type HhshResponse struct {
	Name      string   `json:"name"`
	Trans     []string `json:"trans,omitempty"`
	Inputting []string `json:"inputting,omitempty"`
}

var hhshRe = regexp.MustCompile(`^[a-zA-Z0-9\s]+$`)

func Hhsh(bot *gotgbot.Bot, ctx *ext.Context) error {
	text := getText(ctx)
	_, query := splitCommand(text)
	if query == "" {
		_, err := ctx.Message.Reply(bot, `<a href="https://lab.magiconch.com/nbnhhsh/">能不能好好说话</a>`, &gotgbot.SendMessageOpts{ParseMode: "HTML"})
		return err
	}
	if !hhshRe.MatchString(query) {
		_, err := ctx.Message.Reply(bot, "需要是英文缩写才可以猜测~", nil)
		if err != nil {
			log.Warnf("hhsh reply failed: %s", err)
		}
		return err
	}
	hhshUrl := "https://lab.magiconch.com/nbnhhsh/guess/"
	strings.Fields(query)
	data := url.Values{
		"text": {query},
	}
	resp, err := http.PostForm(hhshUrl, data)
	if err != nil {
		_, err := ctx.Message.Reply(bot, "出现了莫名的网络错误~", nil)
		if err != nil {
			log.Warnf("hhsh reply message failed: %s", err)
			return err
		}
		log.Warnf("post to nbnhhsh website failed: %s", err)
		return err
	}
	defer resp.Body.Close()
	res := make([]HhshResponse, 0)
	read, err := io.ReadAll(resp.Body)
	if err != nil {
		_, err := ctx.Message.Reply(bot, "出现了莫名的响应错误~ 和bot一点关系也没有哦", nil)
		return err
	}
	err = jsoniter.Unmarshal(read, &res)
	if err != nil {
		log.Warnf("hhsh unmarshal failed: %s, data is %s", err, read)
	}
	if len(res) == 0 {
		_, _ = ctx.Message.Reply(bot, "什么也没有猜到~", nil)
	}
	builder := strings.Builder{}
	for _, r := range res {
		if len(r.Trans) == 0 && len(r.Inputting) == 0 {
			builder.WriteString(fmt.Sprintf("%s没找到相关的解释呢~\n", r.Name))
		}
		if len(r.Trans) != 0 {
			builder.WriteString(fmt.Sprintf("%s 找到了:  %s\n", r.Name, strings.Join(r.Trans, ", ")))
		}
		if len(r.Inputting) != 0 {
			builder.WriteString(fmt.Sprintf("%s 可能是:  %s\n", r.Name, strings.Join(r.Inputting, ", ")))
		}
	}
	_, err = ctx.Message.Reply(bot, builder.String(), nil)
	if err != nil {
		log.Warnf("hhsh reply failed: %s", err)
	}
	return err

}
