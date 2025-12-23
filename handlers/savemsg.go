package handlers

import (
	"context"
	"errors"
	g "main/globalcfg"
	"main/globalcfg/msgs"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

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
	if err := saveMessageToMeilisearch(bot, msg); err != nil {
		logD.Error("save message error", zap.Error(err), updateId)
	}
}

func SaveMessage(bot *gotgbot.Bot, ctx *ext.Context) error {
	go saveMessage(bot, ctx)
	return nil
}

func setImageText(bot *gotgbot.Bot, msg *gotgbot.Message, meili *MeiliMsg) error {
	if len(msg.Photo) == 0 {
		return nil
	}
	res, err := ocrMsg(bot, &msg.Photo[len(msg.Photo)-1])
	meili.ImageText = res
	return err
}

func saveMessageToMeilisearch(bot *gotgbot.Bot, msg *gotgbot.Message) (err error) {
	if msg == nil {
		return nil
	}
	uid, err := uuid.NewV7()
	if err != nil {
		return err
	}
	mongoId := uid.String()
	meiliMsg := &MeiliMsg{
		MongoId:   mongoId,
		PeerId:    msg.GetChat().Id,
		FromId:    msg.GetSender().Id(),
		MsgId:     msg.MessageId,
		Date:      float64(msg.Date),
		Message:   msg.GetText(),
		ImageText: "",
		QrResult:  "",
	}
	logger := logD.With(
		zap.Int64("message_id", msg.MessageId),
		zap.Int64("chat_id", msg.GetChat().Id),
		zap.String("mongo_id", mongoId),
	)

	if chatCfg(msg.GetChat().Id).AutoOcr {
		err = setImageText(bot, msg, meiliMsg)
		if err != nil {
			logger.Warn("set image text", zap.Error(err))
		}
	}
	if meiliMsg.Message == "" && meiliMsg.ImageText == "" && meiliMsg.QrResult == "" {
		logger.Debug("skip save message to meilisearch, no text")
		return nil
	}
	err = g.MeiliClient.AddDocument(meiliMsg)
	if err != nil {
		logger.Warn("save message to meilisearch", zap.Error(err))
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
	tx, err := g.RawMsgsDb().BeginTx(context.Background(), pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(context.Background())
		}
	}()
	mdb := g.Msgs.WithTx(tx)
	err = mdb.SaveNewMsg(msg)
	persistRawUpdate(ctx, mdb)
	if err != nil {
		return err
	}
	return tx.Commit(context.Background())
}

func persistRawUpdate(ctx *ext.Context, mdb *msgs.Queries) {
	if g.Msgs == nil {
		return
	}
	if err := mdb.SaveRawUpdate(ctx); err != nil {
		logD.Warn("save raw update failed", zap.Error(err))
	}
}
