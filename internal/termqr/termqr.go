// Package termqr draws a QR code with characters, for a terminal: the panel's remote
// address, Telegram's login code.
package termqr

import (
	"strings"

	"rsc.io/qr"
)

// Text draws text as a QR code with half blocks, two rows of modules per line, for a
// terminal: light modules are drawn and dark ones left blank, as qrencode does, which
// reads right on the usual dark terminal. "" when it cannot be encoded.
func Text(text string) string {
	code, err := qr.Encode(text, qr.L)
	if err != nil {
		return ""
	}
	const quiet = 2
	light := func(x, y int) bool {
		if x < 0 || y < 0 || x >= code.Size || y >= code.Size {
			return true
		}
		return !code.Black(x, y)
	}
	var b strings.Builder
	for y := -quiet; y < code.Size+quiet; y += 2 {
		b.WriteString("  ")
		for x := -quiet; x < code.Size+quiet; x++ {
			top, bottom := light(x, y), light(x, y+1)
			switch {
			case top && bottom:
				b.WriteString("█")
			case top:
				b.WriteString("▀")
			case bottom:
				b.WriteString("▄")
			default:
				b.WriteString(" ")
			}
		}
		b.WriteString("\n")
	}
	return b.String()
}
