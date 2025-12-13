package msgs

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"go.uber.org/zap"
)

const defaultDBTimeout = 2 * time.Second

func nullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}

func marshalJSON(v any) ([]byte, error) {
	if v == nil {
		return nil, nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (q *Queries) marshalWithWarn(v any, field string) []byte {
	js, err := marshalJSON(v)
	if err != nil && q.logger != nil {
		q.logger.Warn("marshal json failed", zap.String("field", field), zap.Error(err))
		return nil
	}
	return js
}

// SaveRawUpdate stores the raw update payload. JSON marshal failure returns an error.
func (q *Queries) SaveRawUpdate(ctx *ext.Context) error {
	if ctx == nil || ctx.Update == nil {
		return nil
	}
	raw, err := json.Marshal(ctx.Update)
	if err != nil {
		return err
	}
	chatID := sql.NullInt64{}
	msgID := sql.NullInt64{}
	if ctx.EffectiveChat != nil {
		chatID = sql.NullInt64{Int64: ctx.EffectiveChat.Id, Valid: true}
	}
	if ctx.EffectiveMessage != nil {
		msgID = sql.NullInt64{Int64: ctx.EffectiveMessage.MessageId, Valid: true}
	}
	c, cancel := context.WithTimeout(context.Background(), defaultDBTimeout)
	defer cancel()
	return q.InsertRawUpdate(c, chatID, msgID, raw)
}

func forwardInfo(msg *gotgbot.Message) (sql.NullString, sql.NullInt64) {
	if msg == nil || msg.ForwardOrigin == nil {
		return sql.NullString{}, sql.NullInt64{}
	}
	fwd := msg.ForwardOrigin.MergeMessageOrigin()
	switch fwd.Type {
	case "user":
		if fwd.SenderUser != nil {
			name := fwd.SenderUser.FirstName
			if fwd.SenderUser.LastName != "" {
				name += " " + fwd.SenderUser.LastName
			}
			return nullString(name), sql.NullInt64{Int64: fwd.SenderUser.Id, Valid: true}
		}
	case "channel":
		if fwd.Chat != nil {
			return nullString(fwd.Chat.Title), sql.NullInt64{Int64: fwd.Chat.Id, Valid: true}
		}
	case "chat":
		if fwd.SenderChat != nil {
			return nullString(fwd.SenderChat.Title), sql.NullInt64{Int64: fwd.SenderChat.Id, Valid: true}
		}
	case "hidden_user":
		return nullString(fwd.SenderUserName), sql.NullInt64{}
	}
	return sql.NullString{}, sql.NullInt64{}
}

func replyInfo(msg *gotgbot.Message) (sql.NullInt64, sql.NullInt64) {
	if msg == nil {
		return sql.NullInt64{}, sql.NullInt64{}
	}
	if msg.ReplyToMessage != nil {
		replyChat := sql.NullInt64{}
		if msg.ReplyToMessage.Chat.Id != 0 {
			replyChat = sql.NullInt64{Int64: msg.ReplyToMessage.Chat.Id, Valid: true}
		}
		return sql.NullInt64{Int64: msg.ReplyToMessage.MessageId, Valid: true}, replyChat
	}
	if msg.ExternalReply != nil {
		replyChat := sql.NullInt64{}
		if msg.ExternalReply.Chat != nil {
			replyChat = sql.NullInt64{Int64: msg.ExternalReply.Chat.Id, Valid: true}
		}
		return sql.NullInt64{Int64: msg.ExternalReply.MessageId, Valid: msg.ExternalReply.MessageId != 0},
			replyChat
	}
	return sql.NullInt64{}, sql.NullInt64{}
}

type mediaInfo struct {
	id   sql.NullString
	uid  sql.NullString
	kind sql.NullString
}

func pickMedia(msg *gotgbot.Message) mediaInfo {
	if msg == nil {
		return mediaInfo{}
	}
	if len(msg.Photo) > 0 {
		last := msg.Photo[len(msg.Photo)-1]
		return mediaInfo{
			id:   nullString(last.FileId),
			uid:  nullString(last.FileUniqueId),
			kind: nullString("photo"),
		}
	}
	if msg.Video != nil {
		return mediaInfo{id: nullString(msg.Video.FileId), uid: nullString(msg.Video.FileUniqueId), kind: nullString("video")}
	}
	if msg.Animation != nil {
		return mediaInfo{id: nullString(msg.Animation.FileId), uid: nullString(msg.Animation.FileUniqueId), kind: nullString("animation")}
	}
	if msg.Document != nil {
		return mediaInfo{id: nullString(msg.Document.FileId), uid: nullString(msg.Document.FileUniqueId), kind: nullString("document")}
	}
	if msg.Audio != nil {
		return mediaInfo{id: nullString(msg.Audio.FileId), uid: nullString(msg.Audio.FileUniqueId), kind: nullString("audio")}
	}
	if msg.Voice != nil {
		return mediaInfo{id: nullString(msg.Voice.FileId), uid: nullString(msg.Voice.FileUniqueId), kind: nullString("voice")}
	}
	if msg.VideoNote != nil {
		return mediaInfo{id: nullString(msg.VideoNote.FileId), uid: nullString(msg.VideoNote.FileUniqueId), kind: nullString("video_note")}
	}
	if msg.Sticker != nil {
		return mediaInfo{id: nullString(msg.Sticker.FileId), uid: nullString(msg.Sticker.FileUniqueId), kind: nullString("sticker")}
	}
	if msg.Story != nil {
		return mediaInfo{
			id:   nullString("story"),
			uid:  nullString("story"),
			kind: nullString("story"),
		}
	}
	return mediaInfo{}
}

func (q *Queries) extraPayload(msg *gotgbot.Message) ([]byte, sql.NullString) {
	if msg == nil {
		return nil, sql.NullString{}
	}
	switch {
	case msg.Contact != nil:
		return q.marshalWithWarn(msg.Contact, "contact"), nullString("contact")
	case msg.Dice != nil:
		return q.marshalWithWarn(msg.Dice, "dice"), nullString("dice")
	case msg.Poll != nil:
		return q.marshalWithWarn(msg.Poll, "poll"), nullString("poll")
	case msg.Venue != nil:
		return q.marshalWithWarn(msg.Venue, "venue"), nullString("venue")
	case msg.Location != nil:
		return q.marshalWithWarn(msg.Location, "location"), nullString("location")
	case msg.Game != nil:
		return q.marshalWithWarn(msg.Game, "game"), nullString("game")
	}
	return nil, sql.NullString{}
}

func (q *Queries) entitiesPayload(msg *gotgbot.Message) []byte {
	var data any
	switch {
	case msg == nil:
		return nil
	case len(msg.Entities) > 0:
		data = msg.Entities
	case len(msg.CaptionEntities) > 0:
		data = msg.CaptionEntities
	default:
		return nil
	}
	return q.marshalWithWarn(data, "entities")
}

// SaveNewMsg inserts a new message row. JSON marshal failures are logged and stored as NULL.
func (q *Queries) SaveNewMsg(msg *gotgbot.Message) error {
	if msg == nil {
		return nil
	}
	forwardName, forwardID := forwardInfo(msg)
	replyMsgID, replyChatID := replyInfo(msg)
	media := pickMedia(msg)
	extra, extraType := q.extraPayload(msg)
	entities := q.entitiesPayload(msg)

	fromUser := sql.NullInt64{}
	if msg.From != nil {
		fromUser = sql.NullInt64{Int64: msg.From.Id, Valid: true}
	}
	senderChat := sql.NullInt64{}
	if msg.SenderChat != nil {
		senderChat = sql.NullInt64{Int64: msg.SenderChat.Id, Valid: true}
	}

	c, cancel := context.WithTimeout(context.Background(), defaultDBTimeout)
	defer cancel()
	viaBot := sql.NullInt64{Valid: msg.ViaBot != nil}
	if viaBot.Valid {
		viaBot.Int64 = msg.ViaBot.Id
	}
	err := q.InsertSavedMessage(c, InsertSavedMessageParams{
		MessageID:         msg.MessageId,
		ChatID:            msg.Chat.Id,
		FromUserID:        fromUser,
		SenderChatID:      senderChat,
		Date:              UnixTime{time.Unix(msg.Date, 0)},
		ForwardOriginName: forwardName,
		ForwardOriginID:   forwardID,
		MessageThreadID:   sql.NullInt64{Int64: msg.MessageThreadId, Valid: msg.MessageThreadId != 0},
		ReplyToMessageID:  replyMsgID,
		ReplyToChatID:     replyChatID,
		ViaBotID:          viaBot,
		EditDate:          sql.Null[UnixTime]{Valid: false},
		MediaGroupID:      nullString(msg.MediaGroupId),
		Text:              nullString(msg.GetText()),
		EntitiesJson:      entities,
		MediaID:           media.id,
		MediaUid:          media.uid,
		MediaType:         media.kind,
		ExtraData:         extra,
		ExtraType:         extraType,
	})
	if err != nil {
		return err
	}
	return nil
}

// SaveEditedMsg updates message text/entities/edit date. JSON marshal failures are logged and stored as NULL.
func (q *Queries) SaveEditedMsg(msg *gotgbot.Message) error {
	if msg == nil || msg.Chat.Id == 0 {
		return nil
	}
	entities := q.entitiesPayload(msg)
	c, cancel := context.WithTimeout(context.Background(), defaultDBTimeout)
	defer cancel()

	err := q.UpdateMessageText(c, UpdateMessageTextParams{
		Text:         nullString(msg.GetText()),
		EntitiesJson: entities,
		EditDate:     sql.Null[UnixTime]{V: UnixTime{time.Unix(msg.EditDate, 0)}, Valid: true},
		ChatID:       msg.Chat.Id,
		MessageID:    msg.MessageId,
	})
	if err != nil {
		return err
	}
	return nil
}
