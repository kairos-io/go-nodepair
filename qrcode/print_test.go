package qrcode

import (
	"image"
	"image/color"
	"io"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/makiuchi-d/gozxing"
	gozxingqr "github.com/makiuchi-d/gozxing/qrcode"
	skipqr "github.com/skip2/go-qrcode"
)

// cell matches one rendered module pair: the foreground colour is the upper
// module, the background colour the lower one.
var cell = regexp.MustCompile(`\x1b\[(\d+);(\d+)m` + upperHalfBlock)

// modulesPerCell is how many pixels wide and high each module becomes when the
// rendered text is turned back into an image. gozxing reads a code at one
// pixel per module, but giving it a few makes the test independent of how the
// decoder samples.
const modulesPerCell = 6

// decodeRendered turns rendered terminal output back into an image and reads
// the QR code in it, the way a camera pointed at the console would.
//
// It is the inverse of Render and knows only the two things Render promises: a
// cell carries the upper module as its foreground and the lower one as its
// background, and a dark module is written with the black codes.
func decodeRendered(t *testing.T, rendered string) string {
	t.Helper()

	var rows [][]bool
	for _, line := range strings.Split(strings.TrimSuffix(rendered, "\n"), "\n") {
		cells := cell.FindAllStringSubmatch(line, -1)
		if len(cells) == 0 {
			t.Fatalf("line %q carries no modules", line)
		}

		upper := make([]bool, len(cells))
		lower := make([]bool, len(cells))
		for x, c := range cells {
			upper[x] = c[1] == "30"
			lower[x] = c[2] == "40"
		}
		rows = append(rows, upper, lower)
	}

	img := image.NewGray(image.Rect(0, 0, len(rows[0])*modulesPerCell, len(rows)*modulesPerCell))
	for y, row := range rows {
		for x, dark := range row {
			shade := color.Gray{Y: 255}
			if dark {
				shade = color.Gray{Y: 0}
			}
			for dy := 0; dy < modulesPerCell; dy++ {
				for dx := 0; dx < modulesPerCell; dx++ {
					img.SetGray(x*modulesPerCell+dx, y*modulesPerCell+dy, shade)
				}
			}
		}
	}

	bmp, err := gozxing.NewBinaryBitmapFromImage(img)
	if err != nil {
		t.Fatalf("building a bitmap from the rendered code: %v", err)
	}

	result, err := gozxingqr.NewQRCodeReader().Decode(bmp, nil)
	if err != nil {
		t.Fatalf("decoding the rendered code: %v", err)
	}

	return result.GetText()
}

// The console output is the only copy of the pairing token an operator ever
// sees, so the test that matters is that a scanner can still read it back.
//
// Every QR version has an odd number of rows once the quiet zone is added, so
// the long token is not about size alone: it is a second symbol version going
// through the same padding of the unpartnered last row.
func TestRenderProducesAScannableCode(t *testing.T) {
	for _, tc := range []struct {
		name  string
		token string
	}{
		{name: "a pairing token", token: "a-pairing-token"},
		{name: "a single character", token: "x"},
		{name: "a token long enough to need a bigger symbol", token: strings.Repeat("long-token-", 20)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			code, err := skipqr.New(tc.token, skipqr.Medium)
			if err != nil {
				t.Fatalf("encoding %q: %v", tc.token, err)
			}

			if got := decodeRendered(t, Render(code.Bitmap())); got != tc.token {
				t.Errorf("the rendered code reads back as %q, want %q", got, tc.token)
			}
		})
	}
}

// A code half as wide as it is high is the shape a scanner rejects, and a code
// taller than the console scrolls its own quiet zone off the screen.
func TestRenderDrawsSquareModulesInHalfTheLines(t *testing.T) {
	code, err := skipqr.New("a-pairing-token", skipqr.Medium)
	if err != nil {
		t.Fatalf("encoding the token: %v", err)
	}

	bitmap := code.Bitmap()
	lines := strings.Split(strings.TrimSuffix(Render(bitmap), "\n"), "\n")

	if want := (len(bitmap) + 1) / 2; len(lines) != want {
		t.Errorf("Render drew %d lines, want %d for a %d row bitmap", len(lines), want, len(bitmap))
	}

	for i, line := range lines {
		if got := len(cell.FindAllString(line, -1)); got != len(bitmap[0]) {
			t.Errorf("line %d carries %d modules, want %d", i, got, len(bitmap[0]))
		}
		if !strings.HasSuffix(line, ansiReset) {
			t.Errorf("line %d does not reset the terminal colours", i)
		}
	}
}

// The quiet zone is what a scanner uses to find the symbol. Rendering it in
// the terminal's own colours instead of an explicit light one leaves it dark
// on a dark theme, which is the one case this rendering exists to avoid.
//
// The bottom border is asserted as well as the top, because it is the one that
// carries the padding of the unpartnered last row.
func TestRenderDrawsTheQuietZoneLight(t *testing.T) {
	code, err := skipqr.New("a-pairing-token", skipqr.Medium)
	if err != nil {
		t.Fatalf("encoding the token: %v", err)
	}

	lines := strings.Split(strings.TrimSuffix(Render(code.Bitmap()), "\n"), "\n")

	for _, border := range []struct {
		name string
		line string
	}{
		{name: "top", line: lines[0]},
		{name: "bottom", line: lines[len(lines)-1]},
	} {
		for i, c := range cell.FindAllStringSubmatch(border.line, -1) {
			if c[1] != "97" || c[2] != "107" {
				t.Fatalf("module %d of the %s border is drawn %s;%s, want a light 97;107", i, border.name, c[1], c[2])
			}
		}
	}
}

// Print is what the callers use, so it has to put the same code on stdout.
func TestPrintWritesTheCodeToStdout(t *testing.T) {
	const token = "a-pairing-token"

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("opening a pipe: %v", err)
	}

	saved := os.Stdout
	os.Stdout = w
	Print(token)
	os.Stdout = saved
	w.Close()

	printed, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("reading what Print wrote: %v", err)
	}

	if got := decodeRendered(t, string(printed)); got != token {
		t.Errorf("Print wrote a code that reads back as %q, want %q", got, token)
	}
}

// There is no QR code for an empty string, and the pairing flow reaches Print
// with one when the token could not be built. Printing a frame with nothing in
// it would look like a code an operator could scan, so Print writes nothing at
// all, the way it did before.
func TestPrintWritesNothingForAnEmptyToken(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("opening a pipe: %v", err)
	}

	saved := os.Stdout
	os.Stdout = w
	Print("")
	os.Stdout = saved
	w.Close()

	printed, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("reading what Print wrote: %v", err)
	}

	if len(printed) != 0 {
		t.Errorf("Print(\"\") wrote %q, want nothing", printed)
	}
}
