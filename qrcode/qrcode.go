package qrcode

import (
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
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

func Print(s string) {
	flagMatte := ""
	var png []byte
	png, err := qrcode.Encode(s, qrcode.Medium, 1)
	if err != nil {
		return
	}
	if flagMatte == "" {
		flagMatte = "000000" // black background
	}
	mc, err := colorful.Hex("#" + flagMatte) // RGB color from Hex format
	if err != nil {
	}

	i, err := ansimage.NewFromReader(strings.NewReader(string(png)), mc, ansimage.NoDithering)
	if err != nil {
	}
	i.DrawExt(false, false)
}

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

func QRCodeReader(s string) (res string) {
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
