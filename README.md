# go-nodepair

A small library to handle transparent remote node pairing.

## Usage

On one side (that is a separate binary, or either a go routine),
we generate a token, display it as QR code and we wait for pairing to complete:


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

    // Generate a token, and print it out on screen as QR code
	t := nodepair.GenerateToken()
	qr.Print(t)

	// bind to the token that we previously generated until we paired with the other remote
    r := map[string]string{}
	nodepair.Receive(ctx, &r, nodepair.WithToken(t))
	fmt.Println("Just got", r)

}

```

Pairing is a _syncronous_ step that waits for the other party to touch base. When pairing completes on both ends, the normal execution flow restarts.

The other party can send a payload which can be any type (`interface{}`).

On the other end:

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

	if err := nodepair.Send(
		ctx, map[string]string{"foo": "Bar"},
		nodepair.WithReader(qr.Reader),
	); err != nil {
		fmt.Println("ERROR", err)
		cancel()
	}
	fmt.Println("Finished sending pairing payload")
}
```

The pairing options can be used in both sender/receiver, allowing to use QR code also for receiving payload, and vice-versa.