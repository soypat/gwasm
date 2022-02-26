package gwasm

import (
	"image/color"
	"math"
	"strconv"
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
