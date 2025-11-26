//go:build frontenddev

package main

import (
	"flag"
	"log"
	"net/http"
	"time"

	g "main/globalcfg"
	"main/http/backend"
)

// compileTime is injected by -ldflags in release builds; keep the same
// symbol name so scripts can reuse it.
var compileTime = "unknown"

func main() {
	listen := flag.String("listen", "127.0.0.1:4021", "fixture backend listen address")
	fixturesPath := flag.String("fixtures", "http/backend/testdata/fixtures.json", "path to fixture JSON")
	authData := flag.String("auth-init-data", "", "Telegram initData for Authorization header validation")
	botToken := flag.String("bot-token", "", "bot token used to verify initData; defaults to fixtures bot_token")
	flag.Parse()

	log.Printf("compile time: %s", compileTime)

	fixtures, err := backend.LoadFixtureData(*fixturesPath)
	if err != nil {
		log.Fatalf("load fixtures: %v", err)
	}

	token := *botToken
	if token == "" {
		token = fixtures.BotToken
	}
	if token == "" {
		log.Fatalf("bot token is required via --bot-token or fixtures.bot_token")
	}
	g.GetConfig().BotToken = token
	backend.ResetBotVerifyKey(token)

	expectedAuth := *authData
	if expectedAuth == "" {
		expectedAuth = fixtures.AuthInitData
	}
	if expectedAuth == "" {
		log.Fatalf("auth init data is required via --auth-init-data or fixtures.auth_init_data")
	}
	if _, err := backend.ParseTelegramAuth(expectedAuth); err != nil {
		log.Fatalf("auth_init_data does not match bot token: %v", err)
	}

	handler, err := backend.NewFixtureHTTPHandler(fixtures, expectedAuth)
	if err != nil {
		log.Fatalf("build fixture handler: %v", err)
	}

	server := &http.Server{
		Addr:              *listen,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("frontenddev fixture backend listening on http://%s", *listen)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("http server stopped: %v", err)
	}
}
