package msgs

import (
	"context"
	"encoding/json"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

const defaultDBTimeout = 2 * time.Second

func textValue(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: s != ""}
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
	if err != nil {
		zap.L().Warn("marshal json failed", zap.String("field", field), zap.Error(err))
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
	chatID := pgtype.Int8{}
	msgID := pgtype.Int8{}
	if ctx.EffectiveChat != nil {
		chatID = pgtype.Int8{Int64: ctx.EffectiveChat.Id, Valid: true}
	}
	if ctx.EffectiveMessage != nil {
		msgID = pgtype.Int8{Int64: ctx.EffectiveMessage.MessageId, Valid: true}
	}
	c, cancel := context.WithTimeout(context.Background(), defaultDBTimeout)
	defer cancel()
	return q.InsertRawUpdate(c, chatID, msgID, raw)
}

func forwardInfo(msg *gotgbot.Message) (pgtype.Text, pgtype.Int8) {
	if msg == nil || msg.ForwardOrigin == nil {
		return pgtype.Text{}, pgtype.Int8{}
	}
	fwd := msg.ForwardOrigin.MergeMessageOrigin()
	switch fwd.Type {
	case "user":
		if fwd.SenderUser != nil {
			name := fwd.SenderUser.FirstName
			if fwd.SenderUser.LastName != "" {
				name += " " + fwd.SenderUser.LastName
			}
			return textValue(name), pgtype.Int8{Int64: fwd.SenderUser.Id, Valid: true}
		}
	case "channel":
		if fwd.Chat != nil {
			return textValue(fwd.Chat.Title), pgtype.Int8{Int64: fwd.Chat.Id, Valid: true}
		}
	case "chat":
		if fwd.SenderChat != nil {
			return textValue(fwd.SenderChat.Title), pgtype.Int8{Int64: fwd.SenderChat.Id, Valid: true}
		}
	case "hidden_user":
		return textValue(fwd.SenderUserName), pgtype.Int8{}
	}
	return pgtype.Text{}, pgtype.Int8{}
}

func replyInfo(msg *gotgbot.Message) (pgtype.Int8, pgtype.Int8) {
	if msg == nil {
		return pgtype.Int8{}, pgtype.Int8{}
	}
	if msg.ReplyToMessage != nil {
		replyChat := pgtype.Int8{}
		if msg.ReplyToMessage.Chat.Id != 0 {
			replyChat = pgtype.Int8{Int64: msg.ReplyToMessage.Chat.Id, Valid: true}
		}
		return pgtype.Int8{Int64: msg.ReplyToMessage.MessageId, Valid: true}, replyChat
	}
	if msg.ExternalReply != nil {
		replyChat := pgtype.Int8{}
		if msg.ExternalReply.Chat != nil {
			replyChat = pgtype.Int8{Int64: msg.ExternalReply.Chat.Id, Valid: true}
		}
		return pgtype.Int8{Int64: msg.ExternalReply.MessageId, Valid: msg.ExternalReply.MessageId != 0},
			replyChat
	}
	return pgtype.Int8{}, pgtype.Int8{}
}

type mediaInfo struct {
	id   pgtype.Text
	uid  pgtype.Text
	kind pgtype.Text
}

func pickMedia(msg *gotgbot.Message) mediaInfo {
	if msg == nil {
		return mediaInfo{}
	}
	if len(msg.Photo) > 0 {
		last := msg.Photo[len(msg.Photo)-1]
		return mediaInfo{
			id:   textValue(last.FileId),
			uid:  textValue(last.FileUniqueId),
			kind: textValue("photo"),
		}
	}
	if msg.Video != nil {
		return mediaInfo{id: textValue(msg.Video.FileId), uid: textValue(msg.Video.FileUniqueId), kind: textValue("video")}
	}
	if msg.Animation != nil {
		return mediaInfo{id: textValue(msg.Animation.FileId), uid: textValue(msg.Animation.FileUniqueId), kind: textValue("animation")}
	}
	if msg.Document != nil {
		return mediaInfo{id: textValue(msg.Document.FileId), uid: textValue(msg.Document.FileUniqueId), kind: textValue("document")}
	}
	if msg.Audio != nil {
		return mediaInfo{id: textValue(msg.Audio.FileId), uid: textValue(msg.Audio.FileUniqueId), kind: textValue("audio")}
	}
	if msg.Voice != nil {
		return mediaInfo{id: textValue(msg.Voice.FileId), uid: textValue(msg.Voice.FileUniqueId), kind: textValue("voice")}
	}
	if msg.VideoNote != nil {
		return mediaInfo{id: textValue(msg.VideoNote.FileId), uid: textValue(msg.VideoNote.FileUniqueId), kind: textValue("video_note")}
	}
	if msg.Sticker != nil {
		return mediaInfo{id: textValue(msg.Sticker.FileId), uid: textValue(msg.Sticker.FileUniqueId), kind: textValue("sticker")}
	}
	if msg.Story != nil {
		return mediaInfo{
			id:   textValue("story"),
			uid:  textValue("story"),
			kind: textValue("story"),
		}
	}
	return mediaInfo{}
}

func (q *Queries) extraPayload(msg *gotgbot.Message) ([]byte, pgtype.Text) {
	if msg == nil {
		return nil, pgtype.Text{}
	}
	switch {
	case msg.Contact != nil:
		return q.marshalWithWarn(msg.Contact, "contact"), textValue("contact")
	case msg.Dice != nil:
		return q.marshalWithWarn(msg.Dice, "dice"), textValue("dice")
	case msg.Poll != nil:
		return q.marshalWithWarn(msg.Poll, "poll"), textValue("poll")
	case msg.Venue != nil:
		return q.marshalWithWarn(msg.Venue, "venue"), textValue("venue")
	case msg.Location != nil:
		return q.marshalWithWarn(msg.Location, "location"), textValue("location")
	case msg.Game != nil:
		return q.marshalWithWarn(msg.Game, "game"), textValue("game")
	}
	return nil, pgtype.Text{}
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

	fromUser := pgtype.Int8{}
	if msg.From != nil {
		fromUser = pgtype.Int8{Int64: msg.From.Id, Valid: true}
	}
	senderChat := pgtype.Int8{}
	if msg.SenderChat != nil {
		senderChat = pgtype.Int8{Int64: msg.SenderChat.Id, Valid: true}
	}

	c, cancel := context.WithTimeout(context.Background(), defaultDBTimeout)
	defer cancel()
	viaBot := pgtype.Int8{Valid: msg.ViaBot != nil}
	if viaBot.Valid {
		viaBot.Int64 = msg.ViaBot.Id
	}
	err := q.InsertSavedMessage(c, InsertSavedMessageParams{
		MessageID:         msg.MessageId,
		ChatID:            msg.Chat.Id,
		FromUserID:        fromUser,
		SenderChatID:      senderChat,
		Date:              pgtype.Timestamptz{Time: time.Unix(msg.Date, 0), Valid: true},
		ForwardOriginName: forwardName,
		ForwardOriginID:   forwardID,
		MessageThreadID:   pgtype.Int8{Int64: msg.MessageThreadId, Valid: msg.MessageThreadId != 0},
		ReplyToMessageID:  replyMsgID,
		ReplyToChatID:     replyChatID,
		ViaBotID:          viaBot,
		EditDate:          pgtype.Timestamptz{},
		MediaGroupID:      textValue(msg.MediaGroupId),
		Text:              textValue(msg.GetText()),
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
		Text:         textValue(msg.GetText()),
		EntitiesJson: entities,
		EditDate:     pgtype.Timestamptz{Time: time.Unix(msg.EditDate, 0), Valid: true},
		ChatID:       msg.Chat.Id,
		MessageID:    msg.MessageId,
	})
	if err != nil {
		return err
	}
	return nil
}
