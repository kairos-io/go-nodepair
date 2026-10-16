package qrcode

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/kbinani/screenshot"
	skipqr "github.com/skip2/go-qrcode"
)

// writeNoCodePNG writes a PNG with no QR code in it, which is what a display
// that is not showing the pairing code captures to.
func writeNoCodePNG(t *testing.T, path string) {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, 300, 300))
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			img.Set(x, y, color.White)
		}
	}

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("creating %s: %v", path, err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatalf("encoding %s: %v", path, err)
	}
}

// writeQRCodePNG writes a PNG carrying token as a QR code.
func writeQRCodePNG(t *testing.T, path, token string) {
	t.Helper()

	if err := skipqr.WriteFile(token, skipqr.Medium, 300, path); err != nil {
		t.Fatalf("encoding %s: %v", path, err)
	}
}

func TestReaderReadsTheCodeInTheFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "code.png")
	writeQRCodePNG(t, path, "a-pairing-token")

	if got := Reader(path); got != "a-pairing-token" {
		t.Errorf("Reader(%q) = %q, want %q", path, got, "a-pairing-token")
	}
}

// The argument is a path to scan, so it is never a token. Handing it back got
// past the empty-token check in PairConfig.Apply, and the path reached edgevpn
// as a network token: `kairosctl register ./not-a-qr.png` failed with
// "illegal base64 data" rather than saying the image carried no code.
func TestReaderDoesNotReturnTheArgumentAsTheToken(t *testing.T) {
	for _, tc := range []struct {
		name string
		path func(t *testing.T) string
	}{
		{
			name: "an image with no code in it",
			path: func(t *testing.T) string {
				p := filepath.Join(t.TempDir(), "blank.png")
				writeNoCodePNG(t, p)
				return p
			},
		},
		{
			name: "a file that is not there",
			path: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "missing.png")
			},
		},
		{
			name: "a file that is not an image",
			path: func(t *testing.T) string {
				p := filepath.Join(t.TempDir(), "notes.txt")
				if err := os.WriteFile(p, []byte("not an image\n"), 0o600); err != nil {
					t.Fatalf("writing %s: %v", p, err)
				}
				return p
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := tc.path(t)

			got := Reader(path)
			if got == path {
				t.Fatalf("Reader(%q) returned its own argument as the token", path)
			}
			// With no display attached the screenshot half cannot succeed
			// either, so the whole read is empty and Apply reports it. A box
			// with a display may legitimately find a code on screen.
			if screenshot.NumActiveDisplays() == 0 && got != "" {
				t.Errorf("Reader(%q) = %q, want an empty token with no displays attached", path, got)
			}
		})
	}
}

func TestReaderWithNoArgumentFallsBackToTheScreen(t *testing.T) {
	if screenshot.NumActiveDisplays() != 0 {
		t.Skip("a display is attached, so what the screen carries is not ours to assert")
	}

	if got := Reader(""); got != "" {
		t.Errorf(`Reader("") = %q, want an empty token with no displays attached`, got)
	}
}
