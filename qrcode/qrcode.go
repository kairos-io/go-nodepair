package qrcode

import (
	"errors"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/kbinani/screenshot"
	"github.com/makiuchi-d/gozxing"
	qr "github.com/makiuchi-d/gozxing/qrcode"

	qrcode "github.com/skip2/go-qrcode"
)

// FromScreenshot reads a QR code from the displays
func FromScreenshot() (string, error) {
	tdir, err := os.MkdirTemp("", "screenshot")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tdir)

	n := screenshot.NumActiveDisplays()
	for i := 0; i < n; i++ {
		bounds := screenshot.GetDisplayBounds(i)

		img, err := screenshot.CaptureRect(bounds)
		if err != nil {
			continue
		}

		os.MkdirAll("/tmp/screenshots", os.ModePerm)
		fileName := fmt.Sprintf(filepath.Join(tdir, "%d_%dx%d.png"), i, bounds.Dx(), bounds.Dy())
		file, _ := os.Create(fileName)
		defer file.Close()
		png.Encode(file, img)
		text, err := Scan(fileName)
		if err == nil && text != "" {
			return text, err
		}
		if err != nil {
			return "", err
		}
	}

	return "", errors.New("nothing found")
}

// upperHalfBlock fills the top half of a terminal cell. A cell is about twice
// as tall as it is wide, so drawing one module per cell gives a code half as
// wide as it is high, which many scanners refuse. Carrying two module rows in
// one cell, the upper in the foreground colour and the lower in the
// background, makes the modules square again and halves the number of lines:
// a 29 by 29 code prints in 15, which fits an 80 by 24 console.
const upperHalfBlock = "▀"

// ansiReset returns the terminal to its own colours at the end of every line,
// so the code does not bleed into whatever is printed after it.
const ansiReset = "\x1b[0m"

// Print prints the string as QRcode on terminal output
func Print(s string) {
	code, err := qrcode.New(s, qrcode.Medium)
	if err != nil {
		return
	}

	fmt.Print(Render(code.Bitmap()))
}

// Render draws a QR code bitmap as lines of terminal text.
//
// bitmap is row-major and true means a dark module. It already carries the
// quiet zone, which is part of the code: a scanner needs the light border to
// find the symbol at all, so it is drawn rather than trimmed.
//
// Both colours are written as explicit ANSI codes instead of being left to the
// terminal's own palette. A scanner reads dark modules on a light background,
// and on a dark theme the untinted output would be the photographic negative
// of that.
func Render(bitmap [][]bool) string {
	var out strings.Builder

	for y := 0; y < len(bitmap); y += 2 {
		upper := bitmap[y]

		// Every QR version has an odd number of rows once the quiet zone is
		// added, so the last row never has a partner. Pad it with light: the
		// half module that adds sits inside the bottom quiet zone, and that
		// zone is light.
		lower := make([]bool, len(upper))
		if y+1 < len(bitmap) {
			lower = bitmap[y+1]
		}

		for x, dark := range upper {
			fmt.Fprintf(&out, "\x1b[%s;%sm%s", foreground(dark), background(lower[x]), upperHalfBlock)
		}

		out.WriteString(ansiReset)
		out.WriteString("\n")
	}

	return out.String()
}

// foreground is the ANSI foreground code for a module, black when it is dark
// and bright white when it is light.
func foreground(dark bool) string {
	if dark {
		return "30"
	}

	return "97"
}

// background is the ANSI background code for a module, black when it is dark
// and bright white when it is light.
func background(dark bool) string {
	if dark {
		return "40"
	}

	return "107"
}

// Scan parses QR code from an image file
func Scan(f string) (string, error) {
	// open and decode image file
	file, err := os.Open(f)
	if err != nil {
		return "", err
	}
	img, _, err := image.Decode(file)
	if err != nil {
		return "", err
	}
	// prepare BinaryBitmap
	bmp, err := gozxing.NewBinaryBitmapFromImage(img)
	if err != nil {
		return "", err
	}
	// decode image
	qrReader := qr.NewQRCodeReader()
	result, err := qrReader.Decode(bmp, nil)
	if err != nil {
		return "", err
	}

	return result.GetText(), nil
}

// Reader retrieves a QRCode from an image file given as argument, or from the
// machine screenshot (best-effort) when there is no file or the file carries
// no code.
//
// It returns the token it read, or an empty string when it read none. s is a
// path to scan, never a token, so it is not a usable fallback: returning it
// got past the empty-token check in PairConfig.Apply and failed much later
// inside edgevpn, with a base64 error that named neither the file nor the
// screen.
//
// Both halves are best-effort, so both say why they came back empty.
func Reader(s string) string {
	if s != "" {
		token, err := Scan(s)
		if err != nil {
			fmt.Printf("warning: %s\n", err.Error())
		}
		if token != "" {
			return token
		}
	}

	token, err := FromScreenshot()
	if err != nil {
		fmt.Printf("warning: %s\n", err.Error())
	}
	return token
}
