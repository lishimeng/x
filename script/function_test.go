package script

import (
	"fmt"
	"reflect"
	"testing"
	"time"
)

func TestReflectSet(t *testing.T) {

	var sample = func(ptr any) {
		tr := reflect.ValueOf(ptr)
		if tr.Kind() != reflect.Ptr || tr.IsNil() {
			return
		}
		t.Log(tr.Elem().Kind())
		if tr.Elem().Kind().String() != "string" {
			t.Log("not string")
			return
		}
		tr.Elem().SetString(fmt.Sprintf("%d", time.Now().Unix()))
	}
	var a string
	sample(&a)
	t.Log(a)
}
