package nodepair_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	nodepair "github.com/kairos-io/go-nodepair"
)

var _ = Describe("Pairing deadlines", func() {
	// A pairing that ran out of clock has to be distinguishable from one that
	// delivered a payload. Returning nil for both makes every caller report a
	// successful install for a node that never showed up.
	It("reports an error when Send's context ends before a node appears", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		err := nodepair.Send(ctx, map[string]string{"foo": "bar"},
			nodepair.WithToken(nodepair.GenerateToken()))

		Expect(err).To(MatchError(context.DeadlineExceeded))
	})

	It("reports an error when Receive's context ends before a payload arrives", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		r := map[string]string{}
		err := nodepair.Receive(ctx, &r, nodepair.WithToken(nodepair.GenerateToken()))

		Expect(err).To(MatchError(context.DeadlineExceeded))
	})

	// The poll loops used a bare time.Sleep, so a finished context was only
	// noticed on the next wake-up: waitNodes sleeps 10s, which is what a
	// caller's deadline was really rounded up to.
	It("gives up close to the deadline rather than at the end of a poll interval", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		start := time.Now()
		err := nodepair.Send(ctx, map[string]string{"foo": "bar"},
			nodepair.WithToken(nodepair.GenerateToken()))

		Expect(err).To(HaveOccurred())
		Expect(time.Since(start)).To(BeNumerically("<", 5*time.Second))
	})

	It("reports an error when Send's context is already cancelled", func() {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := nodepair.Send(ctx, map[string]string{"foo": "bar"},
			nodepair.WithToken(nodepair.GenerateToken()))

		Expect(err).To(HaveOccurred())
	})
})
