package script

import (
	"github.com/robertkrimen/otto"
	"time"
)

type Callback func(name string, params ...ParamStruct) any

type jsEngine struct {
	script           string
	vm               *otto.Otto
	maxExecutionTime time.Duration
}

func (engine *jsEngine) Invoke(method string, params ...interface{}) (otto.Value, error) {
	res, err := CallFunc(engine.vm, method, engine.maxExecutionTime, params...)
	return res, err
}

func (engine *jsEngine) Inject(name string, callback func(call otto.FunctionCall) otto.Value) {
	_ = engine.vm.Set(name, callback)
}

func (engine *jsEngine) InjectFunc(name string, callback Callback, paramType ...ParamType) {

	var cbFunc = func(cb otto.FunctionCall) otto.Value {
		f := Function{
			call: cb,
		}
		f.build(paramType...)
		res := callback(name, f.paramsMap...)
		v, err := otto.ToValue(res)
		if err != nil {
			println(err)
			v = otto.NullValue()
		}
		return v
	}
	engine.Inject(name, cbFunc)
}

func (engine *jsEngine) SetValue(name string, value interface{}) error {
	return engine.vm.Set(name, value)
}
