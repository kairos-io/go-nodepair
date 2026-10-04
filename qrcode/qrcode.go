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
	files := make([]string, 0, n)
	for i := 0; i < n; i++ {
		bounds := screenshot.GetDisplayBounds(i)

		img, err := screenshot.CaptureRect(bounds)
		if err != nil {
			continue
		}

		fileName := filepath.Join(tdir, fmt.Sprintf("%d_%dx%d.png", i, bounds.Dx(), bounds.Dy()))
		if err := writePNG(fileName, img); err != nil {
			continue
		}

		files = append(files, fileName)
	}

	return scanImages(files)
}

// writePNG encodes img into a new file at path.
func writePNG(path string, img image.Image) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return png.Encode(file, img)
}

// scanImages returns the text of the first capture that carries a QR code.
//
// A capture that carries no code decodes to an error rather than to an empty
// string, and so does one that could not be written. Neither says anything
// about the captures after it, so both are skipped and the scan goes on. The
// last of those errors is kept, because it is the only description of why a
// scan that found nothing found nothing.
func scanImages(files []string) (string, error) {
	var lastErr error

	for _, f := range files {
		text, err := Scan(f)
		if err != nil {
			lastErr = err
			continue
		}

		if text != "" {
			return text, nil
		}
	}

	if lastErr != nil {
		return "", fmt.Errorf("nothing found: %w", lastErr)
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
		r, err := Scan(s)
		if err != nil {
			fmt.Printf("warning: %s\n", err.Error())
		}
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
