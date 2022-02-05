package main

import (
	"context"
	"fmt"
	"time"

	nodepair "github.com/mudler/go-nodepair"
	qr "github.com/mudler/go-nodepair/qrcode"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	// First generate a token, that will be read by the other remote via QR code
	t := nodepair.GenerateToken()
	// Prints the pairing token on screen via QR code
	qr.Print(t)

	go func() {
		// On another process send the pairing payload, and read the QR code from screen
		time.Sleep(2 * time.Second)
		fmt.Println("sending payload")
		if err := nodepair.Send(
			ctx, map[string]string{"foo": "Bar"},
			nodepair.WithReader(qr.Reader),
			// Optionally, we could have also supplied a path to a image file, where we could have decoded
			// the qr code.
			nodepair.WithToken(""),
		); err != nil {
			fmt.Println("ERROR", err)
			cancel()
		}
		fmt.Println("Finished sending pairing payload")
	}()

	// Prepare to receive the payload data
	r := map[string]string{}
	// Pair and receive our payload
	nodepair.Receive(ctx, &r, nodepair.WithToken(t))
	// When pairing is completed on both parties, execution flows are restored on both ends.
	fmt.Println(r)
}
