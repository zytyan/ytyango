package myhandlers

import (
	"bytes"
	"encoding/json"
	"io"
	"main/globalcfg"
	"net/http"
	"strings"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/bwmarrin/snowflake"
	"go.uber.org/zap/zapcore"
)

var saveMsgMeili = &http.Client{}
var meili = globalcfg.GetConfig().MeiliConfig
var snowflakeNode = func() *snowflake.Node {
	node, err := snowflake.NewNode(1)
	if err != nil {
		panic(err)
	}
	return node
}

type MeiliMsg struct {
	MongoId   string  `json:"mongo_id"`
	PeerId    int64   `json:"peer_id"`
	FromId    int64   `json:"from_id"`
	MsgId     int64   `json:"msg_id"`
	Date      float64 `json:"date"`
	Message   string  `json:"message,omitempty"`
	ImageText string  `json:"image_text,omitempty"`
	QrResult  string  `json:"qr_result,omitempty"`
}

func SaveMessage(bot *gotgbot.Bot, ctx *ext.Context) error {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Errorf("save message panic %s, update id %d", r, ctx.Update.UpdateId)
			}
		}()
		err := SaveUser(bot, ctx)
		if err != nil {
			log.Errorf("save user error %s, update id %d", err, ctx.Update.UpdateId)
		}
		err = saveMessage(bot, ctx)
		if err != nil {
			log.Errorf("save message error %s, update id %d", err, ctx.Update.UpdateId)
		}
	}()
	return nil
}

type QrRes struct {
	Result []string      `json:"res"`
	Points [][][]float64 `json:"points"`
}

func (q *QrRes) ToString() string {
	return strings.Join(q.Result, "\n\n")
}

func (q *QrRes) Empty() bool {
	return len(q.Result) == 0
}

func setImageText(bot *gotgbot.Bot, msg *gotgbot.Message, meili *MeiliMsg) error {
	if len(msg.Photo) == 0 {
		return nil
	}
	res, err := ocrMsg(bot, &msg.Photo[len(msg.Photo)-1])
	meili.ImageText = res
	return err
}

func saveMessage(bot *gotgbot.Bot, ctx *ext.Context) (err error) {
	if ctx.Message == nil {
		return
	}
	mongoId := snowflakeNode().Generate().Base32()
	meiliMsg := &MeiliMsg{
		MongoId:   mongoId,
		PeerId:    ctx.EffectiveChat.Id,
		FromId:    ctx.EffectiveSender.User.Id,
		MsgId:     ctx.Message.MessageId,
		Date:      float64(ctx.Message.Date),
		Message:   getText(ctx),
		ImageText: "",
		QrResult:  "",
	}
	if g, err := getGroupInfo(ctx.EffectiveChat.Id); err == nil && g.AutoOcr {
		err = setImageText(bot, ctx.Message, meiliMsg)
		if err != nil {
			log.Warnf("set image text error %s, update id %d", err, ctx.Update.UpdateId)
		}
	}
	if meiliMsg.Message == "" && meiliMsg.ImageText == "" && meiliMsg.QrResult == "" {
		log.Debugf("skip save to meilisearch, update id %d", ctx.Update.UpdateId)
		return nil
	}

	j, err := json.Marshal(meiliMsg)
	if err != nil {
		log.Errorf("marshal meili msg error %s, update id %d", err, ctx.Update.UpdateId)
		return err
	}
	log.Debugf("save to meili %s, update id %d", j, ctx.Update.UpdateId)
	preparedPost, _ := http.NewRequest("POST", meili.GetSaveUrl(), bytes.NewReader(j))
	if meili.MasterKey != "" {
		preparedPost.Header.Set("Authorization", "Bearer "+meili.MasterKey)
	}
	preparedPost.Header.Set("Content-Type", "application/json")
	post, err := saveMsgMeili.Do(preparedPost)
	if err != nil {
		log.Infof("post to meili err, update id %d", ctx.Update.UpdateId)
		return err
	}
	defer post.Body.Close()
	var data []byte
	if !(200 <= post.StatusCode && post.StatusCode < 300) {
		data, err = io.ReadAll(post.Body)
		log.Errorf("post to meili err, update id %d, status code %d, response body %s", ctx.Update.UpdateId, post.StatusCode, data)
	}
	if log.Level().Enabled(zapcore.DebugLevel) {
		data, err = io.ReadAll(post.Body)
		log.Debugf("post to meili %s, update id %d", data, ctx.Update.UpdateId)
	}
	return err
}
