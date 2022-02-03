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

	r := map[string]string{}
	t := nodepair.GenerateToken()
	qr.Print(t)

	go func() {
		time.Sleep(2 * time.Second)
		fmt.Println("sending payload")
		if err := nodepair.Send(
			ctx, map[string]string{"foo": "Bar"},
			nodepair.WithReader(qr.Reader),
			nodepair.WithToken(""),
		); err != nil {
			fmt.Println("ERROR", err)
			cancel()
		}
		fmt.Println("Finished sending pairing payload")
	}()
	nodepair.Receive(ctx, &r, nodepair.WithToken(t))
	fmt.Println(r)
}
