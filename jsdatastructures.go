package gwasm

import (
	"image/color"
	"io"
	"math"
	"strconv"
	"syscall/js"
)

// JSColor converts a Go Color to the javascript string
// representation
func JSColor(c color.Color) string {
	r, g, b, a := c.RGBA()
	return "rgb(" + strconv.Itoa(int(r>>8)) + "," +
		strconv.Itoa(int(g>>8)) + "," +
		strconv.Itoa(int(b>>8)) + "," +
		strconv.FormatFloat(float64(a)/math.MaxUint16, 'g', 4, 64) + ")"
}

func console() io.Writer {
	return jsWriter{
		Value: js.Global().Get("console"),
		fname: "log",
	}
}

type jsWriter struct {
	js.Value
	fname string
}

func (j jsWriter) Write(b []byte) (int, error) {
	j.Call(j.fname, string(b))
	return len(b), nil
}
