# go-nodepair

A small library to handle transparent remote node pairing.

> **Found a bug, or want to request a feature?** Open it on
> [kairos-io/kairos](https://github.com/kairos-io/kairos/issues), including
> issues about this repository. Every Kairos issue lives in one place, so you
> never have to work out which repository to file against.

## Usage

On one side (that is a separate binary, or either a go routine),
we generate a token, display it as QR code and we wait for pairing to complete:

```golang

import (
	"context"
	"fmt"
	"time"

	nodepair "github.com/kairos-io/go-nodepair"
	qr "github.com/kairos-io/go-nodepair/qrcode"
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

	nodepair "github.com/kairos-io/go-nodepair"
	qr "github.com/kairos-io/go-nodepair/qrcode"
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
