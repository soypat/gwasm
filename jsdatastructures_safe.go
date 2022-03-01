//go:build safe
// +build safe

package gwasm

import (
	"reflect"
	"syscall/js"
)

// JSTypedArray returns a javascript TypedArray
// for the corresponding Go type.
// See https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/TypedArray
func JSTypedArray(sliceOrArray interface{}) (js.Value, error) {
	TypedArray, _ := typedArrayNameSize(sliceOrArray)
	v := reflect.ValueOf(sliceOrArray)
	len := v.Len()
	dst := js.Global().Get(TypedArray).New(len)
	for i := 0; i < len; i++ {
		dst.SetIndex(i, v.Index(i).Interface())
	}
	return dst, nil
}
