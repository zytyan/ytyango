package genai_hldr

import (
	"context"
	"strings"
	"testing"
	"time"

	"main/globalcfg/q"
)

func TestSplitExecBlocks(t *testing.T) {
	text := "before\n//execjs+\n// summary: add numbers\nconst sum = 1+2;\nreply({sum: sum});\n//execjs-\nafter"
	clean, blocks := splitExecBlocks(text)
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block got %d", len(blocks))
	}
	if strings.Contains(clean, "execjs") {
		t.Fatalf("clean text should remove exec markers, got: %s", clean)
	}
	if blocks[0].Summary != "add numbers" {
		t.Fatalf("unexpected summary: %s", blocks[0].Summary)
	}
	if !strings.Contains(blocks[0].Script, "1+2") {
		t.Fatalf("unexpected script: %s", blocks[0].Script)
	}
}

func TestPromptRendererUsesTemplate(t *testing.T) {
	renderer, err := newPromptRenderer()
	if err != nil {
		t.Fatalf("newPromptRenderer error: %v", err)
	}
	out, err := renderer.Render(PromptData{
		Now:         "2025-01-01",
		ChatType:    "supergroup",
		ChatName:    "测试群",
		BotName:     "Tester",
		BotUsername: "tester_bot",
	})
	if err != nil {
		t.Fatalf("render error: %v", err)
	}
	if !strings.Contains(out, "tester_bot") || !strings.Contains(out, "execjs") {
		t.Fatalf("rendered template missing expected content: %s", out)
	}
}

func TestRunExecBlocksStoresNoteAndReply(t *testing.T) {
	sess := newSession(q.GeminiSession{
		ID:       1,
		ChatID:   123,
		ChatName: "test",
		ChatType: "group",
	}, "bot", "bot_u", 64, 512)
	h := &Handler{
		cfg: Config{
			Exec: ExecLimits{
				Timeout:        time.Second,
				MaxScriptBytes: 1024,
				MaxReplyBytes:  256,
				MaxCallStack:   64,
			},
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	notes, err := h.runExecBlocks(ctx, sess, []execBlock{{
		Summary: "计算 1+1",
		Script:  "reply({v:1+1});",
	}})
	if err != nil {
		t.Fatalf("runExecBlocks error: %v", err)
	}
	if len(notes) != 1 || !strings.Contains(notes[0], "成功") {
		t.Fatalf("unexpected notes: %#v", notes)
	}
	if len(sess.tmpContents) < 2 {
		t.Fatalf("expected note and reply stored, got %d contents", len(sess.tmpContents))
	}
	if strings.Contains(sess.tmpContents[0].Text.String, "reply({v:1+1});") {
		t.Fatalf("note should not expose script content")
	}
}
