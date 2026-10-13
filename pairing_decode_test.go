package nodepair_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	nodepair "github.com/kairos-io/go-nodepair"
)

// halfDecodable fits the first key of the payload the specs below send and not
// the second one, which is what a sender and a receiver that disagree on the
// payload shape look like.
type halfDecodable struct {
	Device string `json:"device"`
	CC     int    `json:"cc"`
}

var _ = Describe("Pairing payloads the receiver cannot decode", func() {
	// Receive dropped the error from Unmarshal and announced "ok" anyway, so a
	// payload that did not fit the caller's value was reported as a delivered
	// pairing to both ends: the caller got nil and a value the failed decode
	// had written, and the sender stopped waiting and said the payload was on
	// its way.
	It("returns the decode error and does not acknowledge the payload", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		token := nodepair.GenerateToken()

		// Buffered, so the sender can finish once the deferred cancel fires
		// without this goroutine outliving the spec.
		sent := make(chan error, 1)
		go func() {
			sent <- nodepair.Send(ctx, map[string]string{"cc": "#cloud-config\n"},
				nodepair.WithToken(token))
		}()

		// The sender announces strings, this caller asks for ints.
		r := map[string]int{}
		err := nodepair.Receive(ctx, &r, nodepair.WithToken(token))

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("decoding the pairing payload"))

		// Not even this one leaves the caller's value alone: the key is
		// created and given the zero value, so a caller checking that the
		// decoded map is non-empty before trusting it still sees a payload.
		Expect(r).To(Equal(map[string]int{"cc": 0}))

		// Receive() here is gomega's channel matcher, not nodepair.Receive.
		// The sender is still waiting, because nothing told it the payload
		// was taken.
		Consistently(sent, "3s", "500ms").ShouldNot(Receive())
	})

	// The dangerous shape is not the payload that fails outright, it is the one
	// that fails halfway: encoding/json records a type error and carries on
	// with the rest of the object, so the caller is handed a value that is
	// filled in part. kairos-agent only checks that the decoded map is not
	// empty, so a half-decoded payload passes that check and installs on the
	// device it did decode with the cloud config it did not.
	It("reports the error when only part of the payload decodes", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		token := nodepair.GenerateToken()

		sent := make(chan error, 1)
		go func() {
			sent <- nodepair.Send(ctx, map[string]string{
				"device": "/dev/sda",
				"cc":     "#cloud-config\n",
			}, nodepair.WithToken(token))
		}()

		p := halfDecodable{}
		err := nodepair.Receive(ctx, &p, nodepair.WithToken(token))

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("decoding the pairing payload"))

		// Proof that this was a partial decode and not a rejected one: the key
		// that did fit is set, and it is the one naming the disk to install on.
		Expect(p.Device).To(Equal("/dev/sda"))
		Expect(p.CC).To(BeZero())
	})
})
