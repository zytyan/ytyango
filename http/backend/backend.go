package backend

import (
	"crypto/sha256"

	g "main/globalcfg"
	api "main/http/backend/ogen"
	"main/myhandlers"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"go.uber.org/zap"
)

// BotProvider returns the bot instance used for authenticated Telegram calls.
type BotProvider func() *gotgbot.Bot

// Handler implements ogen-generated interfaces.
type Handler struct {
	verifyKey   []byte
	botProvider BotProvider
	log         *zap.SugaredLogger
}

// NewHandler builds a handler using the configured bot token hash and bot provider.
func NewHandler(botProvider BotProvider) *Handler {
	if botProvider == nil {
		botProvider = myhandlers.GetMainBot
	}
	sum := sha256.Sum256([]byte(g.GetConfig().BotToken))
	return &Handler{
		verifyKey:   sum[:],
		botProvider: botProvider,
		log:         g.GetLogger("http/backend"),
	}
}

// NewServer wires the ogen server with the backend handler.
func NewServer(botProvider BotProvider) (*api.Server, error) {
	h := NewHandler(botProvider)
	return api.NewServer(h)
}
