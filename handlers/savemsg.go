package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	g "main/globalcfg"
	"main/globalcfg/msgs"
	"runtime/debug"
	"sync/atomic"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/google/uuid"
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

type meiliWALRow struct {
	id      int64
	content string
}

var meiliWALFlushing atomic.Bool

func saveMessage(bot *gotgbot.Bot, ctx *ext.Context) {
	updateFields := []any{"update_id", ctx.UpdateId}
	defer func() {
		if r := recover(); r != nil {
			fields := append(updateFields, "panic", r, "stack", string(debug.Stack()))
			logD.Error("save message panic", fields...)
		}
	}()
	if err := UpdateUser(bot, ctx); err != nil {
		logD.Warn("save user error", append(updateFields, "err", err)...)
	}
	msg := ctx.EffectiveMessage
	if msg == nil {
		return
	}
	if shouldPersistMessages(ctx) {
		if err := persistSavedMessage(ctx, msg); err != nil {
			logD.Warn("persist saved message", append(updateFields, "err", err)...)
		}
	}
	if err := saveMessageToMeilisearch(bot, msg); err != nil {
		logD.Error("save message error", append(updateFields, "err", err)...)
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
		"message_id", msg.MessageId,
		"chat_id", msg.GetChat().Id,
		"mongo_id", mongoId,
	)

	if chatCfg(msg.GetChat().Id).AutoOcr {
		err = setImageText(bot, msg, meiliMsg)
		if err != nil {
			logger.Warn("set image text", "err", err)
		}
	}
	if meiliMsg.Message == "" && meiliMsg.ImageText == "" && meiliMsg.QrResult == "" {
		logger.Debug("skip save message to meilisearch, no text")
		return nil
	}
	content, err := json.Marshal(meiliMsg)
	if err != nil {
		return err
	}
	meiliErr := g.Meili().AddDocument(meiliMsg)
	if meiliErr != nil {
		logger.Warn("save message to meilisearch", "err", meiliErr)
		if err := insertMeiliWAL(context.Background(), g.RawMeiliWalDb(), string(content)); err != nil {
			logger.Warn("save message to meilisearch wal", "err", err)
		}
	} else if err := flushMeiliWAL(context.Background(), g.RawMeiliWalDb(), meiliWALBatchSize(), g.Meili().AddDocuments); err != nil {
		logger.Warn("flush meilisearch wal", "err", err)
	}

	chat := msg.GetChat()
	err = g.Q.UpdateChatAttr(context.Background(), &chat)
	if err != nil {
		logD.Warn("update chat error", "err", err)
		return err
	}
	return meiliErr
}

func meiliWALBatchSize() int {
	cfg := g.GetConfig()
	if cfg == nil || cfg.MeiliWalBatchSize <= 0 {
		return g.DefaultMeiliWalBatchSize
	}
	return cfg.MeiliWalBatchSize
}

func insertMeiliWAL(ctx context.Context, db *sql.DB, content string) error {
	if db == nil {
		return errors.New("meili wal db is nil")
	}
	_, err := db.ExecContext(ctx, `INSERT INTO meili_wal (content) VALUES (?)`, content)
	return err
}

func flushMeiliWAL(ctx context.Context, db *sql.DB, batchSize int, addDocuments func(any) error) error {
	if db == nil {
		return errors.New("meili wal db is nil")
	}
	if addDocuments == nil {
		return errors.New("meili add documents func is nil")
	}
	if batchSize <= 0 {
		batchSize = g.DefaultMeiliWalBatchSize
	}
	if !meiliWALFlushing.CompareAndSwap(false, true) {
		return nil
	}
	defer meiliWALFlushing.Store(false)

	for {
		flushed, err := flushMeiliWALBatch(ctx, db, batchSize, addDocuments)
		if err != nil {
			return err
		}
		if flushed < batchSize {
			return nil
		}
	}
}

func flushMeiliWALBatch(ctx context.Context, db *sql.DB, batchSize int, addDocuments func(any) error) (int, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	rows, err := tx.QueryContext(ctx, `SELECT id, content FROM meili_wal ORDER BY id LIMIT ?`, batchSize)
	if err != nil {
		return 0, err
	}
	walRows, err := readMeiliWALRows(rows)
	if err != nil {
		return 0, err
	}
	if len(walRows) == 0 {
		if err := tx.Commit(); err != nil {
			return 0, err
		}
		committed = true
		return 0, nil
	}

	docs := make([]MeiliMsg, 0, len(walRows))
	for _, row := range walRows {
		var doc MeiliMsg
		if err := json.Unmarshal([]byte(row.content), &doc); err != nil {
			return 0, err
		}
		docs = append(docs, doc)
	}
	if err := addDocuments(docs); err != nil {
		return 0, err
	}
	for _, row := range walRows {
		if _, err := tx.ExecContext(ctx, `DELETE FROM meili_wal WHERE id = ?`, row.id); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	committed = true
	return len(walRows), nil
}

func readMeiliWALRows(rows *sql.Rows) ([]meiliWALRow, error) {
	defer rows.Close()
	var walRows []meiliWALRow
	for rows.Next() {
		var row meiliWALRow
		if err := rows.Scan(&row.id, &row.content); err != nil {
			return nil, err
		}
		walRows = append(walRows, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return walRows, nil
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
		logD.Warn("save raw update failed", "err", err)
	}
}
