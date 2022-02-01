package qrcode

import (
	"errors"
	"fmt"
	"image"
	"image/png"
	"io/ioutil"
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
	tdir, err := ioutil.TempDir("", "screenshot")
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

// Reader retrieves a QRCode or either from an image file
// given as arguments or from the machine screenshot (best-effort)
func Reader(s string) (res string) {
	res = s
	if s == "" {
		res, _ = FromScreenshot()
	} else {
		r, _ := Scan(s)
		if r != "" {
			res = r
		} else {
			r, _ = FromScreenshot()
			if r != "" {
				res = r
			}
		}
	}
	return
}
