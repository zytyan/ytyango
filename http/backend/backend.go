package backend

import (
	"crypto/hmac"
	"crypto/sha256"

	g "main/globalcfg"
	api "main/http/backend/ogen"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"go.uber.org/zap"
)

// Handler implements ogen-generated interfaces.
type Handler struct {
	verifyKey []byte
	bot       *gotgbot.Bot
	log       *zap.Logger
}

// NewHandler builds a handler using the configured bot token hash and bot provider.
func NewHandler(bot *gotgbot.Bot) *Handler {
	mac := hmac.New(sha256.New, []byte("WebAppData"))
	mac.Write([]byte(g.GetConfig().BotToken))
	sum := mac.Sum(nil)
	return &Handler{
		verifyKey: sum,
		bot:       bot,
		log:       g.GetLogger("backend").Desugar(),
	}
}

// NewServer wires the ogen server with the backend handler.
func NewServer(bot *gotgbot.Bot) (*api.Server, error) {
	h := NewHandler(bot)
	return api.NewServer(h, h)
}
