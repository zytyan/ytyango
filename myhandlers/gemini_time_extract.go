package myhandlers

import (
	"context"
	"fmt"
	"main/globalcfg"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	jsoniter "github.com/json-iterator/go"
	"google.golang.org/genai"
)

var getGenAiClient = sync.OnceValues(func() (*genai.Client, error) {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  globalcfg.GetConfig().GeminiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, err
	}
	return client, nil
})
var reTime = regexp.MustCompile(`[一二三四五六七八九十\d][点时日]|\d[:：]\d`)

func hasTimePattern(text string) bool {
	return reTime.MatchString(text)
}
func HasTime(message *gotgbot.Message) bool {
	g, err := getGroupInfo(message.Chat.Id)
	if err != nil {
		return false
	}
	if !g.AutoCalculate {
		return false
	}
	text := message.Text

	if text == "" {
		text = message.Caption
	}
	return hasTimePattern(text)
}
func formatTimezone(off int) string {
	head := "UTC+"
	if off < 0 {
		head = "UTC-"
		off = -off
	}
	minutes := off % 60
	hours := off / 60
	return fmt.Sprintf("%s%02d:%02d", head, hours, minutes)
}
func GeminiExtractTime(bot *gotgbot.Bot, ctx *ext.Context) error {
	user := GetUser(ctx.EffectiveUser.Id)
	if user == nil {
		log.Debugf("user id %d not found", ctx.EffectiveUser.Id)
		return nil
	}
	zoneOffset := 8 * 3600
	if user.TimeZone.Valid {
		zoneOffset = int(user.TimeZone.Int32)
	}
	message := ctx.EffectiveMessage
	text := message.Text

	if text == "" {
		text = message.Caption
	}
	client, err := getGenAiClient()
	if err != nil {
		return err
	}
	genCtx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()
	loc := time.FixedZone("user_timezone", zoneOffset)

	sysInst := "提取下面文字中的时间，以json的格式输出为ISO8601的格式。若文字中没有提到地点或时区，则使用时区 " + formatTimezone(zoneOffset) +
		"若提到了地点或时区 则使用该地点的时区。\n" +
		"例 `[{\"ts\":\"2004-05-03T17:30:08+08:00\"}]`\n" +
		"年月日要符合标准，禁止出现00月或00日 " +
		"若没有可提取的时间信息，则输出空数组`[]`\n" +
		"现在是 " + time.Now().In(loc).Format("2006-01-02T15:04:05-07:00")
	temp := float32(0)
	topK := float32(1)
	config := &genai.GenerateContentConfig{
		SystemInstruction: genai.NewContentFromText(sysInst, genai.RoleUser),
		Temperature:       &temp,
		ResponseMIMEType:  "application/json",
		TopK:              &topK,
	}

	res, err := client.Models.GenerateContent(genCtx, "gemini-2.0-flash-lite", genai.Text(text), config)
	if err != nil {
		return err
	}
	type tsJson struct {
		Ts time.Time `json:"ts"`
	}
	var tsList []tsJson
	err = jsoniter.Unmarshal([]byte(res.Text()), &tsList)
	if err != nil {
		return err
	}
	if len(tsList) == 0 {
		log.Infof("get 0 timestamps")
		return nil
	}
	buf := strings.Builder{}
	buf.WriteString("找到以下时间")
	now := time.Now()
	for _, ts := range tsList {
		buf.WriteByte('\n')
		buf.WriteString(ts.Ts.Format("<code>2006-01-02 15:04:05 -07:00</code>"))
		buf.WriteString("\n  与现在差 ")
		buf.WriteString(ts.Ts.Sub(now).String())
	}
	_, err = ctx.EffectiveMessage.Reply(bot, buf.String(), &gotgbot.SendMessageOpts{
		ParseMode: gotgbot.ParseModeHTML,
	})
	return err
}

func SetUserTimeZone(bot *gotgbot.Bot, ctx *ext.Context) error {
	fields := strings.Fields(ctx.EffectiveMessage.Text)
	const help = "用法: /settimezone +0800"
	if len(fields) < 2 {
		_, err := ctx.EffectiveMessage.Reply(bot, help, nil)
		return err
	}
	t, err := time.Parse("-0700", fields[1])
	if err != nil {
		_, err := ctx.EffectiveMessage.Reply(bot, help, nil)
		return err
	}
	_, zone := t.Zone()
	user := GetUser(ctx.EffectiveUser.Id)
	if user == nil {
		return fmt.Errorf("user id %d not found", ctx.EffectiveUser.Id)
	}
	user.TimeZone.Valid = true
	user.TimeZone.Int32 = int32(zone)
	err = globalcfg.GetDb().Model(user).
		Select("time_zone").
		Updates(user).Error
	if err != nil {
		_, err = ctx.EffectiveMessage.Reply(bot, err.Error(), nil)
		return err
	}
	_, err = ctx.EffectiveMessage.Reply(bot, fmt.Sprintf("设置成功 %d seconds", zone), nil)
	return err
}
