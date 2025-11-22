package h

import "github.com/PaulSonOfLars/gotgbot/v2"

type InlineKeyboardButtonBuilder struct {
	inner [][]gotgbot.InlineKeyboardButton
}

func (b *InlineKeyboardButtonBuilder) Row() *InlineKeyboardButtonBuilder {
	if len(b.inner) == 0 {
		b.inner = append(b.inner, []gotgbot.InlineKeyboardButton{})
	} else {
		back := len(b.inner) - 1
		if len(b.inner[back]) == 0 {
			return b // 没有按钮，不需要新增一行
		}
		b.inner = append(b.inner, []gotgbot.InlineKeyboardButton{})
	}
	return b
}

func (b *InlineKeyboardButtonBuilder) Callback(text, callback string) *InlineKeyboardButtonBuilder {
	if len(b.inner) == 0 {
		b.Row()
	}
	back := len(b.inner) - 1
	b.inner[back] = append(b.inner[back], gotgbot.InlineKeyboardButton{
		Text:         text,
		CallbackData: callback,
	})
	return b
}
