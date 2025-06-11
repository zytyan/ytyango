package mathparser

type VmBuilder struct {
	ByteCode []ByteCode
}

func (vm *VmBuilder) addOp(opCode int, val *Value) {
	vm.ByteCode = append(vm.ByteCode, ByteCode{OpCode: opCode, Val: val})
}
func (vm *VmBuilder) addNoValOp(opCode int) {
	vm.addOp(opCode, nil)
}
func (vm *VmBuilder) addConst(val *Value) {
	vm.addOp(LoadConst, val)
}

func (vm *VmBuilder) Build() *VM {
	return &VM{ByteCode: vm.ByteCode}
}
