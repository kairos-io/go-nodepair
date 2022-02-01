package nodepair_test

import (
	"context"
	"io/ioutil"
	"os"
	"time"

	qr "github.com/mudler/go-nodepair/qrcode"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/skip2/go-qrcode"

	nodepair "github.com/mudler/go-nodepair"
)

var _ = Describe("Pairing", func() {
	token := nodepair.GenerateToken()

	Context("Pairing", func() {
		It("can pair arbitrary data", func() {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			png, err := qrcode.Encode(token, qrcode.Medium, 1)
			if err != nil {
				return
			}
			f, _ := ioutil.TempFile("", "xxx")
			defer os.RemoveAll(f.Name())
			ioutil.WriteFile(f.Name(), png, os.ModePerm)

			go func() {
				time.Sleep(2 * time.Second)
				if err := nodepair.Send(
					ctx, map[string]string{"foo": "Bar"},
					nodepair.WithReader(qr.QRCodeReader),
					nodepair.WithToken(f.Name()),
				); err != nil {
					cancel()
				}
			}()

			r := map[string]string{}
			nodepair.Receive(ctx, &r, nodepair.WithToken(token))

			Expect(r).To(Equal(map[string]string{"foo": "Bar"}))
		})
	})
})
