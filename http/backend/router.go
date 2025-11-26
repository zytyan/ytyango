package backend

import (
	"net/http"
	"time"

	g "main/globalcfg"
	"main/http/backend/botapi"
)

// NewHTTPHandler builds the ogen server with our handler and security implementation.
func NewHTTPHandler() (http.Handler, error) {
	logger := g.GetLogger("http-backend").Desugar()
	handler := &Backend{
		log: logger.Sugar(),
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
	sec := &securityHandler{}
	return botapi.NewServer(handler, sec, botapi.WithErrorHandler(handler.errorHandler))
}
