package cocdice

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewFromText(t *testing.T) {
	as := assert.New(t)
	text := "a\nb\nc"
	br := NewFromText(text)
	as.Equal(br.CurrentCharacter().Name, "a")
	fmt.Println(br.String())
	br.NextCharacter()
	as.Equal(br.CurrentCharacter().Name, "b")
	br.NextCharacter()
	as.Equal(br.CurrentCharacter().Name, "c")
	br.NextCharacter()
	as.Equal(br.CurrentCharacter().Name, "a")
	as.Equal(br.Round, 2)
	br.NextCharacter()
	fmt.Println(br.String())
	err := br.ParseCommand("add d 1 重伤")
	as.Nil(err)
	err = br.ParseCommand("stat 2000 重伤")
	as.Nil(err)
	as.Equal(br.CurrentCharacter().Status, "重伤")

	fmt.Println(br.String())
}
