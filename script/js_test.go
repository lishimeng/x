package script

import (
	"fmt"
	"github.com/robertkrimen/otto"
	"testing"
	"time"
)

func TestJsEngine_Invoke(t *testing.T) {
	testContent := `function decode(fport, data) {
crc("aa", "b")
return {"a": 12, "b": "ffdasf"}
	}`

	var vm, err = Create(testContent)
	if err != nil {
		return
	}
	vm.Inject("crc", func(call otto.FunctionCall) otto.Value {
		if len(call.ArgumentList) < 2 {
			return otto.NullValue()
		}
		p := call.Argument(0)
		pp := call.Argument(1)
		fmt.Println("print param:1")
		fmt.Println(pp.String())
		return p
	})
	raw, err := vm.Invoke("decode", "", "")
	if err != nil {
		return
	}
	ras, err := raw.Export()
	if err != nil {
		return
	}
	switch r := ras.(type) {
	case map[string]interface{}:
		result := r
		fmt.Println(result)
	default:
		fmt.Println("type err")
	}
}

func TestExecute(t *testing.T) {
	testContent := `function decode(fport, data) {
return {"a": 12, "b": "ffdasf"}
	}`
	vm := otto.New()
	raw, err := Execute(vm, testContent, time.Second)
	if err != nil {
		t.Fatal(err)
		return
	}
	t.Log(raw)
}

func TestArray(t *testing.T) {
	t.Log("test return array(string)")
	testContent := `function encode() {
var arr = "123"
arr = arr + "456"
return arr
}`
	vm, err := Create(testContent)
	if err != nil {
		t.Log("create vm fail")
		t.Fatal(err)
		return
	}
	raw, err := vm.Invoke("encode")
	if err != nil {
		t.Log("invoke fail")
		t.Fatal(err)
		return
	}
	t.Log("print result")
	s := raw.String()
	t.Log(s)
}

func TestFunc(t *testing.T) { // 使用crc这个名字注入函数, crc返回校验码
	const testContent = `
function decode(fport, data) {
var p = "00"
p[0] = 0x80
p[1] = 0x05
var payload = "123456"
var s = crc("aa", payload)
return payload + s + "A"
	}
`
	var vm, err = Create(testContent)
	if err != nil {
		return
	}
	var cb = func(name string, params ...ParamStruct) any { // 这里模拟crc产生两个字节校验码
		for index, p := range params {
			var v string
			p.Value(&v)
			println(index, p.Type, v)
		}
		return "5"
	}
	vm.InjectFunc("crc", cb, String, String)
	res, err := vm.Invoke("decode", "", "")

	if err != nil {
		t.Fatal(err)
	}
	t.Log(res.String())
}
