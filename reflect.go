package gwasm

import (
	"fmt"
	"reflect"
	"syscall/js"
)

type JSValuer interface {
	JSValue() js.Value
}

// ValueFromStruct converts a struct with `js` field tags to
// a javascript Object type with the non-nil fields set
// to the struct's values.
// if skipZeroValues is true then fields with a Go zero-value are not
// set in the javascript resulting object.
func ValueFromStruct(Struct interface{}, skipZeroValues bool) js.Value {
	const structTag = "js"
	v := reflect.ValueOf(Struct)
	if v.Kind() == reflect.Ptr && v.IsNil() {
		return js.Null()
	}
	vi := reflect.Indirect(v)
	if vi.Kind() != reflect.Struct {
		panic("expected struct input to ValueFromStruct, got " + vi.Kind().String())
	}
	obj := js.Global().Get("Object").New()
	recordType := vi.Type()
	for i, field := range reflect.VisibleFields(recordType) {
		tag := field.Tag.Get(structTag)
		if tag == "" {
			continue
		}
		fv := vi.Field(i)
		if skipZeroValues && fv.IsZero() {
			// Skip zero values and nil pointers.
			continue
		}
		if fv.Type() == reflect.TypeOf(js.Value{}) {
			obj.Set(tag, fv.Interface().(js.Value))
			continue
		}
		switch field.Type.Kind() {
		case reflect.Float64:
			obj.Set(tag, fv.Float())
		case reflect.String:
			obj.Set(tag, fv.String())
		case reflect.Int:
			obj.Set(tag, fv.Int())
		case reflect.Ptr:
			if fv.IsNil() {
				break
			}
			fv = reflect.Indirect(fv)
			if fv.Kind() != reflect.Struct {
				break
			}
			fallthrough
		case reflect.Struct:
			if fv.NumField() == 0 || fv.Field(0).Type() != reflect.TypeOf(js.Value{}) {
				obj.Set(tag, ValueFromStruct(fv.Interface(), skipZeroValues))
				break
			}
			jsv := fv.Field(0).Interface().(js.Value)
			if jsv.Truthy() {
				obj.Set(tag, jsv)
			}
		case reflect.Interface:
			if ifv, ok := fv.Interface().(JSValuer); ok {
				obj.Set(tag, ifv.JSValue())
			}
		case reflect.Slice:
			arr := js.Global().Get("Array").New()
			for idx := 0; idx < fv.Len(); idx++ {
				sliceVal := fv.Index(idx)
				if sliceVal.Kind() == reflect.Struct {
					arr.Call("push", ValueFromStruct(sliceVal.Interface(), skipZeroValues))
				} else {
					arr.Call("push", sliceVal.Interface())
				}
			}
			obj.Set(tag, arr)
		}
	}
	return obj
}

func typedArrayNameSize(sliceOrArray interface{}) (TypedArray string, sizeOf uintptr) {
	vt := reflect.TypeOf(sliceOrArray)
	if vt.Kind() != reflect.Slice && vt.Kind() != reflect.Array {
		panic("expected slice/array argument to JSTypedArray")
	}
	elem := vt.Elem()
	sizeOf = elem.Size()
	switch elem.String() {
	case "float64":
		TypedArray = "Float64Array"
	case "float32":
		TypedArray = "Float32Array"
	case "int8":
		TypedArray = "Int8Array"
	case "int16":
		TypedArray = "Int16Array"
	case "int32":
		TypedArray = "Int32Array"
	case "int64":
		TypedArray = "BigInt64Array"
	case "uint8":
		TypedArray = "Uint8Array"
	case "uint16":
		TypedArray = "Uint16Array"
	case "uint32":
		TypedArray = "Uint32Array"
	case "uint64":
		TypedArray = "BigUint64Array"
	default:
		panic("unsupported TypedArray Go slice/array type " + vt.Elem().String())
	}
	return TypedArray, sizeOf
}

// Debug prints JSON representation of underlying js.Value if found. Not for use
// with common Go types.
func Debug(a ...interface{}) {
	for _, v := range a {
		fmt.Print(debugs(v) + " ")
	}
	fmt.Println()
}

func stringify(jsv js.Value) string {
	if !jsv.Truthy() {
		return js.Global().Get("String").New(jsv).String()
	}
	return js.Global().Get("JSON").Call("stringify", jsv).String()
}

func debugs(a interface{}) string {
	if s, ok := a.(string); ok {
		return s
	}
	rv := reflect.ValueOf(a)
	if rv.Kind() == reflect.Ptr && rv.IsNil() {
		return "<nil>"
	}
	rv = reflect.Indirect(rv)
	switch {
	case rv.Type() == reflect.TypeOf(js.Value{}):
		// interface is a js.Value.
		return stringify(a.(js.Value))

	case rv.Kind() == reflect.Struct:
		if rv.NumField() == 1 && rv.Field(0).Type() == reflect.TypeOf(js.Value{}) {
			// Single field struct of a js.Value, like most binded types in this package.
			return stringify(rv.Field(0).Interface().(js.Value))
		}
		return stringify(ValueFromStruct(a, false))
	}
	return fmt.Sprintf("%+v", a)
}
