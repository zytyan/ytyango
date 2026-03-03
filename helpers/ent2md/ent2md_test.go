package ent2md

import (
	"testing"

	"github.com/PaulSonOfLars/gotgbot/v2"
)

func TestTgMsgTextToMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		entities []gotgbot.MessageEntity
		expected string
	}{
		{
			name:     "Empty message",
			text:     "",
			entities: nil,
			expected: "",
		},
		{
			name:     "No entities with special characters (Feature)",
			text:     "Hello - world_ *",
			entities: nil,
			// 根据特性：无实体时，原样返回，不做 Markdown 转义
			expected: "Hello - world_ *",
		},
		{
			name: "Basic formatting with trailing text",
			text: "Hello bold and italic text",
			entities: []gotgbot.MessageEntity{
				{Type: "bold", Offset: 6, Length: 4},
				{Type: "italic", Offset: 15, Length: 6},
			},
			// 确保 " text" 这个尾部文本能正确拼接
			expected: "Hello **bold** and _italic_ text",
		},
		{
			name: "Nested entities (Bold inside Italic)",
			text: "This is bold and italic",
			// "bold and italic" is italic, "bold" is also bold
			entities: []gotgbot.MessageEntity{
				{Type: "italic", Offset: 8, Length: 15},
				{Type: "bold", Offset: 8, Length: 4},
			},
			expected: "This is _**bold** and italic_",
		},
		{
			name: "UTF-16 Emoji offset test",
			// "🌍" (Earth Globe Europe-Africa) takes 2 UTF-16 code units.
			// "Hello " is 6 units. "🌍" is 2 units. " " is 1 unit.
			// The word "world" starts at offset 9 in UTF-16, even though it's 8 in runes.
			text: "Hello 🌍 world",
			entities: []gotgbot.MessageEntity{
				{Type: "bold", Offset: 9, Length: 5},
			},
			expected: "Hello 🌍 **world**",
		},
		{
			name: "UTF-16 Multiple Emojis and Complex Characters",
			// "👨‍👩‍👧‍👦" is a complex emoji taking 11 UTF-16 code units.
			// Offset for "family" should be correctly calculated by Telegram.
			// "Hi " (3) + emoji (11) + " " (1) = 15
			text: "Hi 👨‍👩‍👧‍👦 family",
			entities: []gotgbot.MessageEntity{
				{Type: "italic", Offset: 15, Length: 6},
			},
			expected: "Hi 👨‍👩‍👧‍👦 _family_",
		},
		{
			name: "Pre block with language",
			text: "echo 'hello'",
			entities: []gotgbot.MessageEntity{
				{Type: "pre", Offset: 0, Length: 12, Language: "bash"},
			},
			expected: "```bash\necho 'hello'\n```",
		},
		{
			name: "Text Link",
			text: "Go to Google now",
			entities: []gotgbot.MessageEntity{
				{Type: "text_link", Offset: 6, Length: 6, Url: "https://google.com"},
			},
			expected: "Go to [Google](https://google.com) now",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &gotgbot.Message{
				Text:     tt.text,
				Entities: tt.entities,
			}

			result := TgMsgTextToMarkdown(msg)

			if result != tt.expected {
				t.Errorf("\nExpected: %q\nGot:      %q", tt.expected, result)
			}
		})
	}
}
