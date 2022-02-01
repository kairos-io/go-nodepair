package nodepair

import (
	"context"
	"errors"
	"time"

	"github.com/ipfs/go-log"
	libp2p "github.com/libp2p/go-libp2p"
	connmanager "github.com/libp2p/go-libp2p-connmgr"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/mudler/edgevpn/pkg/blockchain"
	"github.com/mudler/edgevpn/pkg/crypto"
	"github.com/mudler/edgevpn/pkg/services"

	"github.com/mudler/edgevpn/pkg/logger"
	node "github.com/mudler/edgevpn/pkg/node"
)

const deadNodes = 20 * time.Minute

func newNode(token string) *node.Node {
	llger := logger.New(log.LevelError)
	mg, _ := connmanager.NewConnManager(20, 100, connmanager.WithGracePeriod(80*time.Second))
	dhtOpts := []dht.Option{dht.BucketSize(20)}
	libp2pOpts := []libp2p.Option{
		libp2p.ConnectionManager(mg),
		libp2p.EnableAutoRelay(),
		libp2p.EnableHolePunching(),
		libp2p.EnableNATService(),
		libp2p.NATPortMap(),
	}

	opts := []node.Option{
		node.WithStore(&blockchain.MemoryStore{}),
		node.WithDiscoveryInterval(10 * time.Second),
		node.WithLedgerAnnounceTime(10 * time.Second),
		node.WithLedgerInterval(10 * time.Second),
		node.Logger(llger),
		node.LibP2PLogLevel(log.LevelError),
		node.WithSealer(&crypto.AESSealer{}),
		node.FromBase64(true, true, token, dhtOpts...),
		node.WithLibp2pOptions(libp2pOpts...),
	}

	return node.New(append(opts, services.Alive(30*time.Second, 5*time.Minute, deadNodes)...)...)
}

// TokenReader is a function that reads a string and returns a token from it.
// A string can represent anything (uri, image file, etc.) which can be used to retrieve the connection token
type TokenReader func(string) string

// PairConfig is the pairing configuration structure
type PairConfig struct {
	tokenReader TokenReader
	token       string
}

// PairOption is a config pair option
type PairOption func(c *PairConfig) error

// WithReader sets the token reader.
// If set, during send is invoked to retrieve a token from the specified string from the client (if any)
func WithReader(t TokenReader) PairOption {
	return func(c *PairConfig) error {
		c.tokenReader = t
		return nil
	}
}

// WithToken sets the token as a pair option
// The token is consumed by TokenReader to parse the string and
// retrieve a token from it.
func WithToken(t string) PairOption {
	return func(c *PairConfig) error {
		c.token = t

		return nil
	}
}

// GenerateToken returns a token which can be used for pairing
func GenerateToken() string {
	d := node.GenerateNewConnectionData()
	return d.Base64()
}

// Receive a payload during pairing
func Receive(ctx context.Context, payload interface{}, opts ...PairOption) error {
	c := &PairConfig{}
	for _, o := range opts {
		if err := o(c); err != nil {
			return err
		}
	}

	n := newNode(c.token)

	if err := n.Start(ctx); err != nil {
		return err
	}

	l, err := n.Ledger()
	if err != nil {
		return err
	}

	waitNodes(ctx, l)

PAIRDATA:
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			v, exists := l.GetKey("pairing", "data")
			if exists {
				v.Unmarshal(payload)
				l.AnnounceUpdate(ctx, 1*time.Second, "pairing", n.Host().ID().String(), "ok")
				break PAIRDATA
			}
			time.Sleep(1 * time.Second)
		}
	}

WAIT:
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			if _, exists := l.GetKey("pairing", n.Host().ID().String()); exists {
				break WAIT
			}
			time.Sleep(1 * time.Second)
		}
	}

	return nil
}

func waitNodes(ctx context.Context, l *blockchain.Ledger) (active []string) {
	active = services.AvailableNodes(l, deadNodes)
	enough := len(active) >= 2
CHECK:
	for !enough {
		select {
		case <-ctx.Done():
			return nil
		default:
			active = services.AvailableNodes(l, deadNodes)
			enough = len(active) >= 2
			if enough {
				break CHECK
			} else {
				time.Sleep(10 * time.Second)
			}
		}
	}

	return active
}

// Send a payload during device pairing
func Send(ctx context.Context, payload interface{}, opts ...PairOption) error {
	c := &PairConfig{}
	for _, o := range opts {
		if err := o(c); err != nil {
			return err
		}
	}

	if c.tokenReader != nil {
		c.token = c.tokenReader(c.token)
	}

	if c.token == "" {
		return errors.New("no token supplied")
	}

	n := newNode(c.token)

	n.Start(ctx)

	l, err := n.Ledger()
	if err != nil {
		return err
	}

	l.AnnounceUpdate(ctx, 1*time.Second, "pairing", "data", payload)

	active := waitNodes(ctx, l)

PAIRING:
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			for _, a := range active {
				if n.Host().ID().String() == a {
					continue
				}
				_, exists := l.GetKey("pairing", a)
				if exists {
					break PAIRING
				}
				time.Sleep(1 * time.Second)
			}
		}
	}
	return nil
}
