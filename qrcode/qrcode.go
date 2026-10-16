package qrcode

import (
	"errors"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	ansimage "github.com/eliukblau/pixterm/pkg/ansimage"
	"github.com/kbinani/screenshot"
	"github.com/lucasb-eyer/go-colorful"
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

// Print prints the string as QRcode on terminal output
func Print(s string) {
	var png []byte
	png, err := qrcode.Encode(s, qrcode.Medium, 1)
	if err != nil {
		return
	}

	mc, err := colorful.Hex("#000000") // RGB color from Hex format
	if err == nil {
		i, err := ansimage.NewFromReader(strings.NewReader(string(png)), mc, ansimage.NoDithering)
		if err == nil {
			i.DrawExt(false, false)
		}
	}
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
