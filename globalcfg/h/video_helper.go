package h

import (
	"context"
	"net/url"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"gopkg.in/vansante/go-ffprobe.v2"
)

func PrepareTgVideo(file string, replyMsgId int64, opts ...func(opts *gotgbot.SendVideoOpts)) (gotgbot.InputFileOrString, *gotgbot.SendVideoOpts) {
	opt := &gotgbot.SendVideoOpts{
		MessageThreadId:       0,
		Duration:              0,
		Width:                 0,
		Height:                0,
		Thumbnail:             nil,
		Cover:                 nil,
		StartTimestamp:        0,
		Caption:               "",
		ParseMode:             "",
		CaptionEntities:       nil,
		ShowCaptionAboveMedia: false,
		HasSpoiler:            false,
		SupportsStreaming:     true,
		DisableNotification:   false,
		ProtectContent:        false,
		AllowPaidBroadcast:    false,
		MessageEffectId:       "",
		ReplyParameters:       nil,
		ReplyMarkup:           nil,
		RequestOpts:           nil,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	probe, err := ffprobe.ProbeURL(ctx, file)
	var vs *ffprobe.Stream
	if err != nil {
		goto skip
	}
	vs = probe.FirstVideoStream()
	if vs != nil {
		if probe.Format != nil {
			opt.Duration = int64(probe.Format.Duration().Seconds())
		}
		opt.Width = int64(vs.Width)
		opt.Height = int64(vs.Height)
	}
skip:
	if replyMsgId > 0 {
		opt.ReplyParameters = &gotgbot.ReplyParameters{
			MessageId:                replyMsgId,
			ChatId:                   0,
			AllowSendingWithoutReply: false,
			Quote:                    "",
			QuoteParseMode:           "",
			QuoteEntities:            nil,
			QuotePosition:            0,
		}
	}
	for _, o := range opts {
		o(opt)
	}
	return gotgbot.InputFileByURL("file://" + url.PathEscape(file)), opt

}
