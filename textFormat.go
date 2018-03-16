package main

import (
	"bytes"
)

const (
	// format set
	formatBold       = 1
	formatDim        = 2
	formatUnderlined = 4
	formatBlink      = 5
	formatReverse    = 7
	formatHidden     = 8

	// format reset
	formatReset           = 0
	formatResetBold       = 21
	formatResetDim        = 22
	formatResetUnderlined = 24
	formatResetBlink      = 25
	formatResetReverse    = 27
	formatResetHidden     = 28

	// color foreground
	foreDefault      = 39
	foreBlack        = 30
	foreRed          = 31
	foreGreen        = 32
	foreYellow       = 33
	foreBlue         = 34
	foreMagenta      = 35
	foreCyan         = 36
	foreLightGray    = 37
	foreDarkGray     = 90
	foreLightRed     = 91
	foreLightGreen   = 92
	foreLightYellow  = 93
	foreLightBlue    = 94
	foreLightMagenta = 95
	foreLightCyan    = 96
	foreWhite        = 97

	// color background
	backDefault      = 49
	backBlack        = 40
	blackRed         = 41
	backGreen        = 42
	backYellow       = 43
	backBlue         = 44
	backMagenta      = 45
	backCyan         = 46
	backLightGray    = 47
	backDarkGray     = 100
	backLightRed     = 101
	backLightGreen   = 102
	backLightYellow  = 103
	backLightBlue    = 104
	backLightMagenta = 105
	backLightCyan    = 106
	backWhite        = 107
)

func FormatTermToHTML(text []byte) string {
	// each text entity in the output is prefixed by 8 bytes indicating the length
	// the sentinel value to look for is STX (start-of-text), 2.

	outBuf := bytes.NewBuffer(nil)

	for i := 0; i < len(text); i++ {
		b := text[i]
		switch b {
		// skip start-of-heading sequence
		// skip start-of-text sequence
		case 1, 2:
			i += 7

		default:
			outBuf.WriteByte(b)
		}
	}

	return outBuf.String()
}
