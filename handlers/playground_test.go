package handlers

import (
	"fmt"
	"testing"

	"github.com/dop251/goja"
)

func runJs(script string) string {
	vm := goja.New()

	v, err := vm.RunString(`let x = {y:100};x.x=x;`)
	v, err = vm.RunString(`JSON.stringify(x)`)
	fmt.Println(v, err)
	return ""
}

func TestMain(m *testing.M) {
	runJs("")
}
