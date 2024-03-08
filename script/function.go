package script

import (
	"github.com/robertkrimen/otto"
	"reflect"
)

type ParamType string

const (
	String  ParamType = "string"  // 字符串/[]byte
	Float   ParamType = "float"   // float64
	Integer ParamType = "integer" // int64
	Boolean ParamType = "boolean" // bool
)

type ParamStruct struct {
	Type  ParamType
	value otto.Value
}

func (p *ParamStruct) Value(ptr any) {
	var err error
	if ptr == nil {
		return
	}
	tr := reflect.ValueOf(ptr)

	if tr.Kind() != reflect.Pointer || tr.IsNil() {
		return
	}
	if p.value.IsUndefined() {
		return
	}

	var elemKind = tr.Elem().Kind()

	switch p.Type {
	case String:
		if elemKind.String() != string(String) {
			return
		}
		var v string
		if p.value.IsString() {
			v, err = p.value.ToString()
			if err != nil {
				return
			}
			tr.Elem().SetString(v)
		}
	case Float:
		if elemKind.String() != "float64" {
			return
		}
		var v float64
		if p.value.IsString() {
			v, err = p.value.ToFloat()
			if err != nil {
				return
			}
			tr.Elem().SetFloat(v)
		}
	case Integer:
		if elemKind.String() != "int64" {
			return
		}
		var v int64
		if p.value.IsNumber() {
			v, err = p.value.ToInteger()
			if err != nil {
				return
			}
			tr.Elem().SetInt(v)
		}
	case Boolean:
		if elemKind.String() != "bool" {
			return
		}
		var v bool
		if p.value.IsBoolean() {
			v, err = p.value.ToBoolean()
			if err != nil {
				return
			}
			tr.Elem().SetBool(v)
		}
	}
}

type Function struct {
	call      otto.FunctionCall
	paramsMap []ParamStruct
}

func (f *Function) build(paramTypes ...ParamType) {
	for i, paramType := range paramTypes {
		var p = ParamStruct{
			Type: paramType,
		}
		var value = f.call.Argument(i)
		p.value = value
		f.paramsMap = append(f.paramsMap, p)
	}
}

func (f *Function) GetParam(i int) *ParamStruct {
	if len(f.paramsMap) <= i {
		return nil
	}
	return &f.paramsMap[i]
}
