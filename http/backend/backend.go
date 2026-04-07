package backend

import (
	"crypto/hmac"
	"crypto/sha256"
	"log"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"

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

func slogGinLogger(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		attrs := []slog.Attr{
			slog.String("method", c.Request.Method),
			slog.String("path", path),
			slog.Int("status", c.Writer.Status()),
			slog.Duration("latency", time.Since(start)),
			slog.String("client_ip", c.ClientIP()),
			slog.String("user_agent", c.Request.UserAgent()),
		}
		if rawQuery := c.Request.URL.RawQuery; rawQuery != "" {
			attrs = append(attrs, slog.String("query", rawQuery))
		}
		if len(c.Errors) > 0 {
			attrs = append(attrs, slog.String("errors", c.Errors.String()))
		}

		level := slog.LevelInfo
		if c.Writer.Status() >= http.StatusInternalServerError {
			level = slog.LevelError
		} else if c.Writer.Status() >= http.StatusBadRequest {
			level = slog.LevelWarn
		}
		log.LogAttrs(c.Request.Context(), level, "gin request", attrs...)
	}
}

// NewServer wires the gin router with the backend handler.
func NewServer(bot *gotgbot.Bot) (http.Handler, error) {
	registerValidators()
	h := NewHandler(bot)
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(slogGinLogger(h.log), gin.Recovery())
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
