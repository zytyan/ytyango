package cocdice

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
)

const (
	NormalDice = iota
	BonusDice
	PenaltyDice
)

type DiceCommand struct {
	Type     int
	Arg1     int
	Arg2     int
	Modifier int
	Ability  int
}

func Map[T, U any](ts []T, f func(T) U) []U {
	us := make([]U, len(ts))
	for i := range ts {
		us[i] = f(ts[i])
	}
	return us
}

func Sum(arr []int) int {
	var s int
	for _, v := range arr {
		s += v
	}
	return s
}

func minIndex(a []int) int {
	idx := 0
	for i, v := range a {
		if v < a[idx] {
			idx = i
		}
	}
	return idx
}

func maxIndex(a []int) int {
	idx := 0
	for i, v := range a {
		if v > a[idx] {
			idx = i
		}
	}
	return idx
}

func (d *DiceCommand) formatAbility(points int) string {
	if d.Ability == 0 {
		return ""
	}
	var data [3]int
	data[0] = d.Ability
	data[1] = int(float64(d.Ability) * 0.5)
	data[2] = int(float64(d.Ability) * 0.2)
	if points < data[2] {
		return fmt.Sprintf("\n (%d / %d / <b><u>%d</u></b>)", data[0], data[1], data[2])
	} else if points < data[1] {
		return fmt.Sprintf("\n (%d / <b><u>%d</u></b> / %d)", data[0], data[1], data[2])
	} else if points < data[0] {
		return fmt.Sprintf("\n (<b><u>%d</u></b> / %d / %d)", data[0], data[1], data[2])
	}
	return fmt.Sprintf("\n (%d / %d / %d)", data[0], data[1], data[2])
}

func (d *DiceCommand) normalDice() string {
	if d.Arg1 > 100 {
		return "骰子数量太多了"
	}
	dices := make([]int, d.Arg1)
	for i := 0; i < d.Arg1; i++ {
		dices[i] = rand.Intn(d.Arg2) + 1
	}
	sum := Sum(dices)
	sum += d.Modifier
	text := strings.Builder{}
	text.WriteString(fmt.Sprintf("骰子点数: %d", sum))
	if d.Modifier != 0 {
		text.WriteString(fmt.Sprintf(" (%+d)", d.Modifier))
	}
	text.WriteString(d.formatAbility(sum))
	if d.Arg1 > 1 {
		text.WriteString("\n")
		text.WriteString(strings.Join(Map(dices, strconv.Itoa), " + "))
	}
	return text.String()
}
func (d *DiceCommand) bonusOrPenaltyDice(name string, idxFunc func([]int) int) string {
	count := d.Arg1 + 1
	dices := make([]int, count)
	for i := 0; i < count; i++ {
		dices[i] = rand.Intn(10)
	}
	onesPlace := rand.Intn(10) + 1
	minIdx := idxFunc(dices)
	sum := dices[minIdx]*10 + onesPlace

	sum += d.Modifier
	text := strings.Builder{}
	text.WriteString(fmt.Sprintf("%s点数: %d", name, sum))
	if d.Modifier != 0 {
		text.WriteString(fmt.Sprintf(" (%+d)", d.Modifier))
	}
	text.WriteString(d.formatAbility(sum))

	tensStr := formatTensIdx(dices, minIdx)
	text.WriteString(fmt.Sprintf("\n个位骰：%d\n十位骰：%s", onesPlace, tensStr))
	return text.String()
}
func (d *DiceCommand) bonusDice() string {
	if d.Arg1 > 100 {
		return "奖励骰子数量太多了"
	}
	return d.bonusOrPenaltyDice("奖励骰", minIndex)
}
func (d *DiceCommand) penaltyDice() string {
	if d.Arg1 > 100 {
		return "惩罚骰子数量太多了"
	}
	return d.bonusOrPenaltyDice("惩罚骰", maxIndex)
}
func (d *DiceCommand) Roll() string {
	switch d.Type {
	case NormalDice:
		return d.normalDice()
	case BonusDice:
		return d.bonusDice()
	case PenaltyDice:
		return d.penaltyDice()
	}
	return "出现错误"
}
func formatTensIdx(tens []int, idx int) string {
	var buf []string
	for i, tensPlace := range tens {
		if i == idx {
			buf = append(buf, fmt.Sprintf("<b><u>[%d]</u></b>", tensPlace))
		} else {
			buf = append(buf, fmt.Sprintf("<del>%d</del>", tensPlace))
		}

	}
	tensPlaces := strings.Join(buf, ", ")
	return fmt.Sprintf("%s", tensPlaces)
}
