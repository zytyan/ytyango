package myhandlers

import (
	"bytes"
	"fmt"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	jsoniter "github.com/json-iterator/go"
	"html"
	"net/http"
	"regexp"
	"time"
)

type TimeResp struct {
	Text   string `json:"text"`
	Offset []int  `json:"offset"`
	Type   string `json:"type"`
	Detail struct {
		Type       string `json:"type"`
		Definition string `json:"definition"`

		Time jsoniter.RawMessage `json:"time"`
	} `json:"detail"`
}

func (t *TimeResp) GetTime() (time.Time, error) {
	data := t.Detail.Time
	i := 0
	for data[i] == ' ' {
		i++
	}
	data = data[i:]
	switch data[0] {
	case '[':
		var s []string
		_ = jsoniter.Unmarshal(data, &s)
		ti, err := time.ParseInLocation(time.DateTime, s[0], time.Local)
		return ti, err
	default:
		return time.Time{}, fmt.Errorf("unknown time format %s", data)
	}
}

var reRemindMsg = regexp.MustCompile(`提醒`)

func IsRemindMsg(msg *gotgbot.Message) bool {
	if group, err := getGroupInfo(msg.Chat.Id); err == nil && group != nil && group.AutoCalculate {
		return reRemindMsg.MatchString(msg.Text)
	}
	return false
}

func getRemindTime(msg *gotgbot.Message) (time.Time, error) {
	text := getTextMsg(msg)
	j, err := jsoniter.Marshal(map[string]string{
		"text": text,
	})
	if err != nil {
		return time.Time{}, err
	}
	resp, err := http.Post("http://localhost:8000/parse_time", "application/json", bytes.NewReader(j))
	if err != nil {
		return time.Time{}, err
	}
	defer resp.Body.Close()
	var timeResps []TimeResp
	err = jsoniter.NewDecoder(resp.Body).Decode(&timeResps)
	if err != nil {
		return time.Time{}, err
	}
	for _, timeResp := range timeResps {
		if timeResp.Type == "time_point" {
			t, err := timeResp.GetTime()
			if err != nil {
				log.Warnf("get time from %s failed: %s", timeResp.Detail.Time, err)
				continue
			}
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("找不到提醒的时间")
}

func getRemindText(msg *gotgbot.Message) (string, error) {
	return getTextMsg(msg), nil
}

func Remind(bot *gotgbot.Bot, ctx *ext.Context) (err error) {
	msg := ctx.EffectiveMessage
	t, err := getRemindTime(msg)
	if err != nil {
		_, _ = msg.Reply(bot, err.Error(), nil)
		return
	}
	text, err := getRemindText(msg)
	if err != nil {
		_, _ = msg.Reply(bot, "找不到提醒的时间", nil)
		return
	}
	replyNow := fmt.Sprintf("好的，bot会在 %s 提醒你。\n请注意，当前功能为测试功能，目前无法持久化提醒，可能会随时间改变。", t.Format(time.RFC3339))
	_, err = msg.Reply(bot, replyNow, nil)
	if err != nil {
		return
	}
	go func() {
		log.Infof("remind %s at %s", text, t)
		time.Sleep(time.Until(t))
		log.Infof("remind %s now", text)
		remindUser := msg.From.Id
		text = html.EscapeString(fmt.Sprintf("提醒 %s", text))
		htmlTxt := fmt.Sprintf("<a href=\"tg://user?id=%d\">%s</a>", remindUser, text)
		_, err = bot.SendMessage(msg.Chat.Id, htmlTxt, &gotgbot.SendMessageOpts{ParseMode: gotgbot.ParseModeHTML})
		if err != nil {
			log.Warnf("send remind msg failed: %s", err)
		}
	}()
	return
}
