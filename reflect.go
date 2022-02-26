package gwasm

import (
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
		panic("expected struct input to objectify")
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
				arr.Call("push", ValueFromStruct(fv.Index(idx).Interface(), skipZeroValues))
			}
			obj.Set(tag, arr)
		}
	}
	return obj
}
