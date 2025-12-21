package backend

import (
	"crypto/hmac"
	"crypto/sha256"
	"log"
	"net/http"

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
	_, _ = mac.Write([]byte(g.GetConfig().BotToken))
	sum := mac.Sum(nil)
	return &Handler{
		verifyKey: sum,
		bot:       bot,
		log:       g.GetLogger("http-backend", zap.WarnLevel).Desugar(),
	}
}

// NewServer wires the ogen server with the backend handler.
func NewServer(bot *gotgbot.Bot) (*api.Server, error) {
	h := NewHandler(bot)
	return api.NewServer(h, h)
}

func ListenAndServe(addr string, bot *gotgbot.Bot) error {
	server, err := NewServer(bot)
	if err != nil {
		return err
	}
	err = http.ListenAndServe(addr, server)
	if err != nil {
		return err
	}
	return nil
}

func GoListenAndServe(addr string, bot *gotgbot.Bot) {
	go func() {
		err := ListenAndServe(addr, bot)
		if err != nil {
			log.Fatal(err)
		}
	}()
}
