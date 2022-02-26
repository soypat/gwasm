//go:build !safe
// +build !safe

package gwasm

import (
	"errors"
	"reflect"
	"syscall/js"
	"unsafe"
)

// JSTypedArray returns a javascript TypedArray
// for the corresponding Go type.
// See https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/TypedArray
func JSTypedArray(slice interface{}) (js.Value, error) {
	v := reflect.ValueOf(slice)
	vt := reflect.TypeOf(slice)
	if vt.Kind() != reflect.Slice {
		panic("expected slice argument to JSTypedArray")
	}
	size := vt.Elem().Size()

	var TypedArray string
	switch v.Elem().Interface().(type) {
	case *float64:
		TypedArray = "Float64Array"
	case *float32:
		TypedArray = "Float32Array"
	case *int8:
		TypedArray = "Int8Array"
	case *int16:
		TypedArray = "Int16Array"
	case *int32:
		TypedArray = "Int32Array"
	case *int64:
		TypedArray = "BigInt64Array"
	case *uint8:
		TypedArray = "Uint8Array"
	case *uint16:
		TypedArray = "Uint16Array"
	case *uint32:
		TypedArray = "Uint32Array"
	case *uint64:
		TypedArray = "BigUint64Array"
	default:
		panic("unsupported slice type")
	}
	// Get the slice header
	header := *(*reflect.SliceHeader)(unsafe.Pointer(v.UnsafeAddr()))

	// We obtain a backing slice in bytes.
	header.Len *= int(size)
	header.Cap *= int(size)

	// Convert slice header to an []byte
	src := *(*[]byte)(unsafe.Pointer(&header))

	dst := js.Global().Get("Uint8Array").New(header.Len)
	n := js.CopyBytesToJS(dst, src)
	if n != len(src) {
		return dst, errors.New("TypedArray write unsuccesful")
	}
	if TypedArray == "Uint8Array" {
		return dst, nil
	}
	return js.Global().Get(TypedArray).New(dst.Get("buffer")), nil
}
