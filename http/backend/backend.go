package backend

import (
	"crypto/hmac"
	"crypto/sha256"
	"log"
	"log/slog"
	"net/http"
	"strconv"
	"sync"

	g "main/globalcfg"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// Handler implements ogen-generated interfaces.
type Handler struct {
	verifyKey []byte
	bot       *gotgbot.Bot
	log       *slog.Logger
}

var registerValidatorOnce sync.Once

// NewHandler builds a handler using the configured bot token hash and bot provider.
func NewHandler(bot *gotgbot.Bot) *Handler {
	mac := hmac.New(sha256.New, []byte("WebAppData"))
	_, _ = mac.Write([]byte(g.GetConfig().BotToken))
	sum := mac.Sum(nil)
	return &Handler{
		verifyKey: sum,
		bot:       bot,
		log:       g.GetLogger("http-backend", slog.LevelWarn),
	}
}

func registerValidators() {
	registerValidatorOnce.Do(func() {
		validate, ok := binding.Validator.Engine().(*validator.Validate)
		if !ok {
			return
		}
		_ = validate.RegisterValidation("int64str", func(fl validator.FieldLevel) bool {
			_, err := strconv.ParseInt(fl.Field().String(), 10, 64)
			return err == nil
		})
	})
}

// NewServer wires the gin router with the backend handler.
func NewServer(bot *gotgbot.Bot) (http.Handler, error) {
	registerValidators()
	h := NewHandler(bot)
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.POST("/search", h.requireHeaderAuth("SearchMessages"), h.handleSearchMessages)
	r.POST("/users/info", h.requireHeaderAuth("GetUsersInfo"), h.handleGetUsersInfo)
	r.GET("/users/:userId/avatar", h.handleGetUserAvatar)
	return r, nil
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
