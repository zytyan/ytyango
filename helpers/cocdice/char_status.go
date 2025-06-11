package cocdice

import (
	"errors"
	"fmt"
	"html"
	"slices"
	"strconv"
	"strings"
)

type BattleRound struct {
	Round      int
	Current    int
	Characters []*Character
}

type Character struct {
	Name   string
	Order  int
	Status string
}

func (b *BattleRound) CurrentCharacter() *Character {
	return b.Characters[b.Current]
}

func (b *BattleRound) String() string {
	var result strings.Builder
	result.WriteString(fmt.Sprintf("第%d回合\n\n", b.Round))
	for i, c := range b.Characters {
		var header = " "
		if i == b.Current {
			header = "&gt;"
		}
		result.WriteString(fmt.Sprintf("<code>%d %s</code>%s: [%s]\n",
			c.Order, header,
			html.EscapeString(c.Name),
			html.EscapeString(c.Status)))
	}
	result.WriteString("使用 add [名字] [顺序] [状态] 添加角色\n")
	result.WriteString("使用 chg [旧顺序] [新顺序] 修改顺序\n")
	result.WriteString("使用 del [名字/顺序] 删除角色\n")
	result.WriteString("使用 stat [名字/顺序] [状态] 修改状态\n")
	return result.String()
}

func (b *BattleRound) NextCharacter() {
	b.Current++
	if b.Current >= len(b.Characters) {
		b.Current = 0
		b.Round++
	}
}
func (b *BattleRound) findCharIdxByName(name string) int {
	for i, c := range b.Characters {
		if c.Name == name {
			return i
		}
	}
	return -1
}

func (b *BattleRound) findCharIdxByOrder(order int) int {
	for i, c := range b.Characters {
		if c.Order == order {
			return i
		}
	}
	return -1
}

func (b *BattleRound) AddCharacter(name string, order int, status string) error {
	for _, c := range b.Characters {
		if c.Name == name {
			return fmt.Errorf("角色 %s 已经存在", name)
		}
		if c.Order == order {
			return fmt.Errorf("顺序 %d 已经存在", order)
		}
	}
	b.Characters = append(b.Characters, &Character{Name: name, Order: order, Status: status})
	slices.SortFunc(b.Characters, func(i, j *Character) int {
		return i.Order - j.Order
	})
	idx := b.findCharIdxByOrder(order)
	if idx == -1 {
		return nil
	}
	if idx <= b.Current {
		b.Current++
	}
	return nil
}
func (b *BattleRound) deleteCharByIdx(idx int) error {
	b.Characters = append(b.Characters[:idx], b.Characters[idx+1:]...)
	if b.Current > idx {
		b.Current--
	}
	if b.Current >= len(b.Characters) || b.Current < 0 {
		b.Current = 0
	}
	return nil
}
func (b *BattleRound) DeleteCharacter(order int) error {
	idx := b.findCharIdxByOrder(order)
	if idx == -1 {
		return errors.New("找不到顺序 " + strconv.Itoa(order))
	}
	return b.deleteCharByIdx(idx)
}

func (b *BattleRound) DeleteCharacterByName(name string) error {
	idx := b.findCharIdxByName(name)
	if idx == -1 {
		return errors.New("找不到名字 " + name)
	}
	return b.deleteCharByIdx(idx)
}

func (b *BattleRound) SetCharacterStatusByOrder(order int, status string) error {
	idx := b.findCharIdxByOrder(order)
	if idx != -1 {
		b.Characters[idx].Status = status
		return nil
	}
	return fmt.Errorf("找不到顺序 %d", order)
}

func (b *BattleRound) SetCharacterStatusByName(name string, status string) error {
	idx := b.findCharIdxByName(name)
	if idx != -1 {
		b.Characters[idx].Status = status
		return nil
	}
	return fmt.Errorf("找不到名字 %s", name)
}
func (b *BattleRound) ChangeCharacterOrder(oldOrder, newOrder int) error {
	oldIdx := b.findCharIdxByOrder(oldOrder)
	if oldIdx != -1 {
		newIdx := b.findCharIdxByOrder(newOrder)
		if newIdx != -1 {
			return fmt.Errorf("顺序 %d 已经存在", newOrder)
		}
		b.Characters[oldIdx].Order = newOrder
		slices.SortFunc(b.Characters, func(i, j *Character) int {
			return i.Order - j.Order
		})
		return nil
	}
	return fmt.Errorf("找不到顺序 %d", oldOrder)
}

func (b *BattleRound) ParseCommand(command string) error {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return fmt.Errorf("空命令")
	}
	switch strings.ToLower(parts[0]) {
	case "add":
		if len(parts) < 4 {
			return fmt.Errorf("用法: add [名字] [顺序] [状态]")
		}
		name := parts[1]
		order, _ := strconv.Atoi(parts[2])
		status := parts[3]
		return b.AddCharacter(name, order, status)
	case "chg", "change":
		if len(parts) < 3 {
			return fmt.Errorf("用法: change [旧顺序] [新顺序]")
		}
		oldOrder, e1 := strconv.Atoi(parts[1])
		newOrder, e2 := strconv.Atoi(parts[2])
		if e1 != nil || e2 != nil {
			return fmt.Errorf("顺序必须是数字，但是你输入了 %s %s", parts[1], parts[2])
		}
		return b.ChangeCharacterOrder(oldOrder, newOrder)
	case "del":
		if len(parts) < 2 {
			return fmt.Errorf("用法: del [名字/顺序]")
		}
		order, e := strconv.Atoi(parts[1])
		if e == nil {
			return b.DeleteCharacter(order)

		}
		return b.DeleteCharacterByName(parts[1])
	case "stat":
		if len(parts) < 3 {
			return fmt.Errorf("用法: stat [名字/顺序] [状态]")
		}
		order, e := strconv.Atoi(parts[1])
		if e == nil {
			return b.SetCharacterStatusByOrder(order, parts[2])
		}
		return b.SetCharacterStatusByName(parts[1], parts[2])
	default:
		return fmt.Errorf("未知命令 %s", parts[0])
	}
}

func NewFromText(text string) *BattleRound {
	br := &BattleRound{Round: 1}
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "/") {
			continue
		}
		_ = br.AddCharacter(line, (i+1)*1000, "正常")
	}
	br.Current = 0
	return br
}
