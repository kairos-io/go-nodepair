package qrcode

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestQRCode(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "QRCode Suite")
}
