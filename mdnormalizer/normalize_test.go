package mdnormalizer

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeBasicFormatting(t *testing.T) {
	msg, err := Normalize("Hello **world** 😄")
	require.NoError(t, err)
	require.Equal(t, "Hello world 😄\n", msg.Text)
	require.Len(t, msg.Entities, 1)

	ent := msg.Entities[0]
	require.Equal(t, "bold", ent.Type)
	require.EqualValues(t, 6, ent.Offset)
	require.EqualValues(t, 5, ent.Length)
}

func TestNormalizeCodeBlock(t *testing.T) {
	msg, err := Normalize("```\ncode\n```")
	require.NoError(t, err)
	require.Equal(t, "code\n\n", msg.Text)
	require.Len(t, msg.Entities, 1)

	ent := msg.Entities[0]
	require.Equal(t, "pre", ent.Type)
	require.EqualValues(t, 0, ent.Offset)
	require.EqualValues(t, utf16Length("code\n"), ent.Length)
}

func TestNormalizeListFallback(t *testing.T) {
	msg, err := Normalize("- a\n- b")
	require.NoError(t, err)
	require.Equal(t, "• a\n• b\n", msg.Text)
	require.Empty(t, msg.Entities)
	require.Empty(t, msg.Warnings)
}

func TestNormalizeMathFallback(t *testing.T) {
	msg, err := Normalize("Equation $E=mc^2$ done.")
	require.NoError(t, err)

	require.Equal(t, "Equation E=mc^2 done.\n", msg.Text)
	require.Len(t, msg.Entities, 1)
	ent := msg.Entities[0]
	require.Equal(t, "code", ent.Type)
	require.EqualValues(t, utf16Length("Equation "), ent.Offset)
	require.EqualValues(t, utf16Length("E=mc^2"), ent.Length)
	require.Contains(t, msg.Warnings, "math converted to inline code")
}

func TestNormalizeImageFallback(t *testing.T) {
	msg, err := Normalize("![alt](https://ex.com/img.png)")
	require.NoError(t, err)

	require.Equal(t, "alt\n", msg.Text)
	require.Len(t, msg.Entities, 1)
	ent := msg.Entities[0]
	require.Equal(t, "text_link", ent.Type)
	require.Equal(t, "https://ex.com/img.png", ent.Url)
	require.Contains(t, msg.Warnings, "image converted to link")
}

func TestNormalizeEmojiOffset(t *testing.T) {
	msg, err := Normalize("👋 **hi**")
	require.NoError(t, err)
	require.Equal(t, "👋 hi\n", msg.Text)
	require.Len(t, msg.Entities, 1)
	ent := msg.Entities[0]
	require.EqualValues(t, utf16Length("👋 "), ent.Offset)
	require.EqualValues(t, utf16Length("hi"), ent.Length)
}

func TestMarkdownText(t *testing.T) {
	text := "这里有一段Markdown文案，你可以拿去测试：\n\n" +
		"# 这是一个一级标题\n\n" +
		"## 这是一个二级标题\n\n" +
		"**粗体文字** 和 *斜体文字*。\n\n" +
		"这是一个有序列表：\n" +
		"1. 第一项\n" +
		"2. 第二项\n\n" +
		"这是一个无序列表：\n" +
		"- 项目 A\n" +
		"- 项目 B\n\n" +
		"这是一个 [链接](https://example.com)。\n\n" +
		"这是一个代码块：\n" +
		"```javascript\n" +
		"function test() {\n" +
		"  console.log(\"Hello Markdown!\");\n" +
		"}\n" +
		"```" +
		"\n\n希望这段文案对你的测试有用！"
	msg, err := Normalize(text)
	require.NoError(t, err)
	expected := `这里有一段Markdown文案，你可以拿去测试：
这是一个一级标题
这是一个二级标题
粗体文字 和 斜体文字。
这是一个有序列表：
1. 第一项
2. 第二项
这是一个无序列表：
• 项目 A
• 项目 B
这是一个 链接。
这是一个代码块：
function test() {
  console.log("Hello Markdown!");
}

希望这段文案对你的测试有用！`
	require.Equal(t, expected+"\n", msg.Text)
}

func TestEscape(t *testing.T) {
	r := require.New(t)
	msg, err := Normalize(`\.`)
	r.NoError(err)
	r.Equal("\\.\n", msg.Text)
	r.Empty(msg.Entities)
}
