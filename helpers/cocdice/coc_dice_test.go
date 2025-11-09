package cocdice

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDiceCommand_NormalDice1(t *testing.T) {
	as := assert.New(t)
	diceCommand := DiceCommand{Type: NormalDice, Arg1: 1, Arg2: 100, Modifier: 5, Ability: 70}
	result := diceCommand.normalDice()
	fmt.Println(result)
	as.Contains(result, "70")
	as.Contains(result, "35")
	as.Contains(result, "14")
	as.Equal(strings.Count(result, "+"), 1)
}
func TestDiceCommand_NormalDice2(t *testing.T) {
	as := assert.New(t)
	diceCommand := DiceCommand{Type: NormalDice, Arg1: 3, Arg2: 100, Modifier: 5, Ability: 70}
	result := diceCommand.normalDice()
	fmt.Println(result)
	as.Contains(result, "70")
	as.Contains(result, "35")
	as.Contains(result, "14")
	as.Equal(strings.Count(result, "+"), 3)
}
func TestDiceCommand_BonusDice(t *testing.T) {
	as := assert.New(t)
	diceCommand := DiceCommand{Type: BonusDice, Arg1: 1, Arg2: 100, Modifier: 0, Ability: 70}
	result := diceCommand.bonusDice()
	fmt.Println(result)
	as.Contains(result, "70")
	as.Contains(result, "35")
	as.Contains(result, "14")
	as.Equal(strings.Count(result, "+"), 0)
}
