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
func JSTypedArray(sliceOrArray interface{}) (js.Value, error) {
	TypedArray, size := typedArrayNameSize(sliceOrArray)
	v := reflect.ValueOf(sliceOrArray)
	if v.Len() == 0 {
		return js.Global().Get(TypedArray).New(0), nil
	}

	// Get the slice header
	header := reflect.SliceHeader{
		Data: v.Index(0).UnsafeAddr(),
		// We obtain a backing slice in bytes so length is multiple of size.
		Len: v.Len() * int(size),
		Cap: v.Cap() * int(size),
	}

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
