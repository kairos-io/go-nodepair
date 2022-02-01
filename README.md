# go-nodepair

A small library to handle transparent remote node pairing.

## Usage

```golang
import (
	"context"
	"fmt"
	"time"

	nodepair "github.com/mudler/go-nodepair"
	qr "github.com/mudler/go-nodepair/qrcode"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

    // Generate a token, and print it out somehow
	t := nodepair.GenerateToken()
	qr.Print(t)

	go func() {
        // Now in another process, read the QR code and use it to 
        // send a payload to the other remote end
        // Note: This can be a separate binary
		time.Sleep(2 * time.Second)
		if err := nodepair.Send(
			ctx, map[string]string{"foo": "Bar"},
			nodepair.WithReader(qr.Reader),
			nodepair.WithToken(""), 
            // Optionally provide an input to the QRCode reader. This can be a png file, or the token in the text form. 
            // The reader will take a screenshot as a fallback
		); err != nil {
			fmt.Println("ERROR", err)
			cancel()
		}
		fmt.Println("Finished sending pairing payload")
	}()

    // bind to the token that we previously generated until we paired with the other remote
    r := map[string]string{}
	nodepair.Receive(ctx, &r, nodepair.WithToken(t))
	fmt.Println("Just got", r)
}
```