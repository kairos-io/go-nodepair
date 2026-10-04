package qrcode

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	skipqr "github.com/skip2/go-qrcode"
)

// writeBlank writes a PNG with no QR code in it, which is what a display that
// is not showing the pairing code captures to.
func writeBlank(path string) {
	img := image.NewRGBA(image.Rect(0, 0, 300, 300))
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			img.Set(x, y, color.White)
		}
	}

	f, err := os.Create(path)
	Expect(err).ToNot(HaveOccurred())
	defer f.Close()
	Expect(png.Encode(f, img)).To(Succeed())
}

var _ = Describe("scanImages", func() {
	var dir, blank, code, missing string

	BeforeEach(func() {
		var err error
		dir, err = os.MkdirTemp("", "qrcode-test")
		Expect(err).ToNot(HaveOccurred())

		blank = filepath.Join(dir, "blank.png")
		code = filepath.Join(dir, "code.png")
		missing = filepath.Join(dir, "never-written.png")

		writeBlank(blank)
		Expect(skipqr.WriteFile("pairing-token", skipqr.Medium, 256, code)).To(Succeed())
	})

	AfterEach(func() {
		Expect(os.RemoveAll(dir)).To(Succeed())
	})

	// A display that carries no code decodes to a NotFoundException, not to an
	// empty string, so treating that error as fatal ends the scan on the first
	// display and the code on any later one is never seen.
	It("keeps scanning after a capture that carries no code", func() {
		Expect(scanImages([]string{blank, code})).To(Equal("pairing-token"))
	})

	// A capture that could not be written at all has to be skipped for the same
	// reason: it says nothing about the displays after it.
	It("keeps scanning after a capture that was never written", func() {
		Expect(scanImages([]string{missing, code})).To(Equal("pairing-token"))
	})

	It("returns the code when it is on the first capture", func() {
		Expect(scanImages([]string{code, blank})).To(Equal("pairing-token"))
	})

	It("reports an error when no capture carries a code", func() {
		text, err := scanImages([]string{blank, missing})
		Expect(text).To(BeEmpty())
		Expect(err).To(MatchError(ContainSubstring("nothing found")))
	})

	It("reports an error when there is nothing to scan", func() {
		text, err := scanImages(nil)
		Expect(text).To(BeEmpty())
		Expect(err).To(MatchError(ContainSubstring("nothing found")))
	})
})

var _ = Describe("writePNG", func() {
	var dir string

	BeforeEach(func() {
		var err error
		dir, err = os.MkdirTemp("", "qrcode-test")
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		Expect(os.RemoveAll(dir)).To(Succeed())
	})

	It("writes an image a scan can read back", func() {
		path := filepath.Join(dir, "out.png")
		img, err := skipqr.New("written-token", skipqr.Medium)
		Expect(err).ToNot(HaveOccurred())
		Expect(writePNG(path, img.Image(256))).To(Succeed())

		Expect(Scan(path)).To(Equal("written-token"))
	})

	// A capture that cannot be created has to report it rather than be passed
	// on to the scan as a file that is not there. png.Encode fails on the nil
	// file as well, so this holds the outcome rather than the line that
	// produces it.
	It("reports an error when the file cannot be created", func() {
		Expect(writePNG(filepath.Join(dir, "no-such-dir", "out.png"), image.NewRGBA(image.Rect(0, 0, 1, 1)))).
			ToNot(Succeed())
	})
})
