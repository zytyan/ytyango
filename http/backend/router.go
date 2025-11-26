package backend

import (
	"net/http"
	"time"

	g "main/globalcfg"
	"main/http/backend/botapi"
)

const allowedOrigin = "http://localhost:5173"

// wrapWithCORS adds a narrow CORS policy for local frontend dev server.
func wrapWithCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == allowedOrigin || origin == allowedOrigin+"/" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

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
	server, err := botapi.NewServer(handler, sec, botapi.WithErrorHandler(handler.errorHandler))
	if err != nil {
		return nil, err
	}
	return wrapWithCORS(server), nil
}

// NewFixtureHTTPHandler builds a server backed by generated fixtures.
// It is used by the frontenddev build tag to expose deterministic data
// without talking to Telegram or the SQLite database.
func NewFixtureHTTPHandler(fixtures FixtureData, expectedAuth string) (http.Handler, error) {
	handler, err := NewFixtureBackend(fixtures, expectedAuth)
	if err != nil {
		return nil, err
	}
	sec := &securityHandler{}
	server, err := botapi.NewServer(handler, sec, botapi.WithErrorHandler(handler.errorHandler))
	if err != nil {
		return nil, err
	}
	return wrapWithCORS(server), nil
}
