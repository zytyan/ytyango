package replacer

import (
	"testing"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
)

func testCtx(chat gotgbot.Chat) *ReplaceCtx {
	return &ReplaceCtx{
		Bot: &gotgbot.Bot{User: gotgbot.User{FirstName: "Mars", LastName: "Bot", Username: "marsbot"}},
		Msg: &gotgbot.Message{Chat: chat},
		Now: time.Date(2024, 3, 14, 15, 9, 26, 0, time.UTC),
	}
}

func TestReplacerEmptyTemplate(t *testing.T) {
	r := NewReplacer("")
	if got := r.Replace(nil); got != "" {
		t.Fatalf("expected empty result, got %q", got)
	}
}

func TestReplacerEscapesAndLiterals(t *testing.T) {
	tpl := "rate 100%% ok"
	r := NewReplacer(tpl)
	if got := r.Replace(testCtx(gotgbot.Chat{})); got != "rate 100% ok" {
		t.Fatalf("unexpected replace result: %q", got)
	}

	tpl = "%%TIME%"
	r = NewReplacer(tpl)
	if got := r.Replace(testCtx(gotgbot.Chat{})); got != "%TIME%" {
		t.Fatalf("unexpected escape result: %q", got)
	}

	tpl = "hello %TIME"
	r = NewReplacer(tpl)
	if got := r.Replace(testCtx(gotgbot.Chat{})); got != "hello %TIME" {
		t.Fatalf("unexpected unclosed result: %q", got)
	}
}

func TestReplacerUnknownAndInvalidVars(t *testing.T) {
	ctx := testCtx(gotgbot.Chat{})
	tpl := "%UNKNOWN% %chat_name% %A-B% %TIME%"
	r := NewReplacer(tpl)
	if got := r.Replace(ctx); got != "%UNKNOWN% %chat_name% %A-B% 15:09:26" {
		t.Fatalf("unexpected result: %q", got)
	}
}

func TestReplacerMetaVars(t *testing.T) {
	ctx := testCtx(gotgbot.Chat{Title: "Test Group"})
	tpl := "chat:%CHAT_NAME% bot:%BOT_NAME% user:%BOT_USERNAME% time:%TIME% date:%DATE% dt:%DATETIME%"
	r := NewReplacer(tpl)
	if got := r.Replace(ctx); got != "chat:Test Group bot:Mars Bot user:marsbot time:15:09:26 date:2024-03-14 dt:2024-03-14 15:09:26" {
		t.Fatalf("unexpected result: %q", got)
	}

	ctx = testCtx(gotgbot.Chat{FirstName: "Alice", LastName: "Doe", Username: "alice"})
	tpl = "chat:%CHAT_NAME%"
	r = NewReplacer(tpl)
	if got := r.Replace(ctx); got != "chat:Alice Doe" {
		t.Fatalf("unexpected chat name: %q", got)
	}
}
