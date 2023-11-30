package mathparser

import (
	"fmt"
	"runtime/debug"
)

//goland:noinspection ALL
const (
	typeNumber = "number"
	typeStr    = "string"
)

//goland:noinspection ALL
const (
	Exit      = 0xffff
	LoadConst = iota
	LoadVar
	LoadFunc

	SaveVar

	UnaryAdd
	UnarySub
	UnaryMul
	UnaryDiv
	UnaryMod
	UnaryPow
	UnaryDice
	UnaryFactorial
	UnaryPerm
	UnaryComb

	UnaryNot
	UnaryGreater
	UnaryLess
	UnaryEqual

	CallFunc
	Return

	Jump
	JumpIfTrue
)

type Value struct {
	valType string
	Num     *number
}

func (v *Value) ToText() string {
	if v.valType == typeNumber {
		return v.Num.toText()
	}
	return ""
}

type ByteCode struct {
	OpCode int
	Val    *Value
}

type VM struct {
	ByteCode []ByteCode
	stack    []Value

	callStack []int
	PC        int

	VarMap      map[string]Value
	VarMapStack []map[string]Value

	err error
}

func (vm *VM) push(val Value) {
	vm.stack = append(vm.stack, val)
}
func (vm *VM) pop() Value {
	if len(vm.stack) == 0 {
		return Value{}
	}
	last := len(vm.stack) - 1
	res := vm.stack[last]
	vm.stack = vm.stack[:last]
	return res
}
func (vm *VM) error(err string) {
	debug.PrintStack()
	vm.err = fmt.Errorf(err)
}

func (vm *VM) loadConst(val Value) {

	vm.push(val)
}

func (vm *VM) add() {
	right := vm.pop()
	left := vm.pop()
	if left.valType == typeNumber && right.valType == typeNumber {
		vm.push(Value{valType: typeNumber, Num: left.Num.add(right.Num)})
	} else {
		vm.error("not implemented")
	}
}
func (vm *VM) sub() {

	right := vm.pop()
	left := vm.pop()
	if left.valType == typeNumber && right.valType == typeNumber {
		vm.push(Value{valType: typeNumber, Num: left.Num.sub(right.Num)})
	} else {
		vm.error("not implemented")
	}
}
func (vm *VM) mul() {

	right := vm.pop()
	left := vm.pop()
	if left.valType == typeNumber && right.valType == typeNumber {
		vm.push(Value{valType: typeNumber, Num: left.Num.mul(right.Num)})
	} else {
		vm.error("not implemented")
	}
}
func (vm *VM) div() {

	right := vm.pop()
	left := vm.pop()
	if left.valType == typeNumber && right.valType == typeNumber {
		vm.push(Value{valType: typeNumber, Num: left.Num.div(right.Num)})
	} else {
		vm.error("not implemented")
	}
}
func (vm *VM) mod() {
	right := vm.pop()
	left := vm.pop()
	if left.valType == typeNumber && right.valType == typeNumber {
		vm.push(Value{valType: typeNumber, Num: left.Num.mod(right.Num)})
	} else {
		vm.error("not implemented")
	}
}

func (vm *VM) pow() {

	right := vm.pop()
	left := vm.pop()
	if left.valType == typeNumber && right.valType == typeNumber {
		vm.push(Value{valType: typeNumber, Num: left.Num.pow(right.Num)})
	} else {
		vm.error("not implemented")
	}
}
func (vm *VM) checkSameType(opNum int) bool {
	if opNum == 1 {
		return true
	}
	if len(vm.stack) < opNum {
		vm.err = fmt.Errorf("stack size is not enough")
		return false
	}
	for i := 0; i < opNum-1; i++ {
		op1 := vm.stack[len(vm.stack)-1-i]
		op2 := vm.stack[len(vm.stack)-2-i]
		if op1.valType != op2.valType {
			vm.err = fmt.Errorf("type not match, type1: %s, type2: %s", op1.valType, op2.valType)
			return false
		}
	}
	return true
}

func (vm *VM) runCode() {
	for vm.PC < len(vm.ByteCode) {
		op := vm.ByteCode[vm.PC]
		switch op.OpCode {
		case LoadConst:
			vm.loadConst(*op.Val)
		case UnaryAdd:
			vm.add()
		case UnarySub:
			vm.sub()
		case UnaryMul:
			vm.mul()
		case UnaryDiv:
			vm.div()
		case UnaryMod:
			vm.mod()
		case UnaryPow:
			vm.pow()
		default:
			vm.err = fmt.Errorf("unknown opcode: %d", op.OpCode)
		}
		if vm.err != nil {
			return
		}
		vm.PC++
	}
}
func (vm *VM) Run() (res *Value, err error) {
	vm.runCode()
	if vm.err != nil {
		return nil, vm.err
	}
	if len(vm.stack) == 0 {
		return nil, nil
	}
	return &vm.stack[len(vm.stack)-1], nil
}
