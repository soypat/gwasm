package gwasm_test

import (
	"testing"

	"github.com/soypat/gwasm"
)

func TestJSTypedArray(t *testing.T) {
	a := []uint8{1, 2, 4, 255}
	val, err := gwasm.JSTypedArray(a)
	if err != nil {
		t.Fatal(err)
	}
	for i := range a {
		if a[i] != uint8(val.Index(i).Int()) {
			t.Error("value mismatch")
		}
	}

	b := []float32{0, .125, 1. / 3, 1000}
	val, err = gwasm.JSTypedArray(b)
	if err != nil {
		t.Fatal(err)
	}
	for i := range b {
		if b[i] != float32(val.Index(i).Float()) {
			t.Error("value mismatch")
		}
	}
}
