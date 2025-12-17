package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	g "main/globalcfg"
	"main/globalcfg/msgs"
	"net/http"
	"strings"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/bwmarrin/snowflake"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var saveMsgMeili = &http.Client{}
var meili = g.GetConfig().MeiliConfig
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

func saveMessage(bot *gotgbot.Bot, ctx *ext.Context) {
	updateId := zap.Int64("update_id", ctx.UpdateId)
	defer func() {
		if r := recover(); r != nil {
			logD.Error("save message panic",
				zap.Any("panic", r),
				zap.Stack("stack"),
				updateId)
		}
	}()
	if err := UpdateUser(bot, ctx); err != nil {
		logD.Warn("save user error", zap.Error(err), updateId)
	}
	msg := ctx.EffectiveMessage
	if msg == nil {
		return
	}
	if shouldPersistMessages(ctx) {
		if err := persistSavedMessage(ctx, msg); err != nil {
			logD.Warn("persist saved message", zap.Error(err), updateId)
		}
	}
	if err := saveMessageToMeilisearch(bot, ctx, msg); err != nil {
		logD.Error("save message error", zap.Error(err), updateId)
	}
}

func SaveMessage(bot *gotgbot.Bot, ctx *ext.Context) error {
	go saveMessage(bot, ctx)
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

func saveMessageToMeilisearch(bot *gotgbot.Bot, ctx *ext.Context, msg *gotgbot.Message) (err error) {
	if msg == nil {
		return nil
	}
	mongoId := snowflakeNode().Generate().Base32()
	meiliMsg := &MeiliMsg{
		MongoId:   mongoId,
		PeerId:    ctx.EffectiveChat.Id,
		FromId:    msg.GetSender().Id(),
		MsgId:     msg.MessageId,
		Date:      float64(msg.Date),
		Message:   getText(ctx),
		ImageText: "",
		QrResult:  "",
	}
	logger := logD.With(
		zap.Int64("update_id", ctx.UpdateId),
		zap.Int64("message_id", msg.MessageId),
		zap.Int64("chat_id", ctx.EffectiveChat.Id),
		zap.String("mongo_id", mongoId),
	)

	if chatCfg(ctx.EffectiveChat.Id).AutoOcr {
		err = setImageText(bot, msg, meiliMsg)
		if err != nil {
			logger.Warn("set image text", zap.Error(err))
		}
	}
	if meiliMsg.Message == "" && meiliMsg.ImageText == "" && meiliMsg.QrResult == "" {
		logger.Debug("skip save message to meilisearch, no text")
		return nil
	}

	j, err := json.Marshal(meiliMsg)
	if err != nil {
		logger.Error("marshal meili msg", zap.Error(err))
		return err
	}
	logger.Debug("save to meili", zap.ByteString("json", j))
	preparedPost, _ := http.NewRequest("POST", meili.GetSaveUrl(), bytes.NewReader(j))
	if meili.MasterKey != "" {
		preparedPost.Header.Set("Authorization", "Bearer "+meili.MasterKey)
	}
	preparedPost.Header.Set("Content-Type", "application/json")
	post, err := saveMsgMeili.Do(preparedPost)
	if err != nil {
		logger.Warn("post to meili", zap.Error(err))
		return err
	}
	defer post.Body.Close()
	var data []byte
	logFn := logger.Debug

	if !(200 <= post.StatusCode && post.StatusCode < 300) || logger.Level().Enabled(zapcore.DebugLevel) {
		if !(200 <= post.StatusCode && post.StatusCode < 300) {
			logFn = logger.Error
		}
		data, err = io.ReadAll(post.Body)
		logFn("post to meili",
			zap.Int("status_code", post.StatusCode),
			zap.ByteString("response", data),
			zap.NamedError("read_body_error", err),
		)
	}

	return err
}

func shouldPersistMessages(ctx *ext.Context) bool {
	cfg := g.GetConfig()
	if cfg == nil || !cfg.SaveMessage {
		return false
	}
	if ctx.EffectiveChat == nil {
		return false
	}
	return chatCfg(ctx.EffectiveChat.Id).SaveMessages
}

func persistSavedMessage(ctx *ext.Context, msg *gotgbot.Message) error {
	if g.Msgs == nil {
		return errors.New("msgs querier is nil")
	}
	tx, err := g.RawMsgsDb().BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	mdb := g.Msgs.WithTx(tx)
	err = mdb.SaveNewMsg(msg)
	persistRawUpdate(ctx, mdb)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func persistRawUpdate(ctx *ext.Context, mdb *msgs.Queries) {
	if g.Msgs == nil {
		return
	}
	if err := mdb.SaveRawUpdate(ctx); err != nil {
		logD.Warn("save raw update failed", zap.Error(err))
	}
}
