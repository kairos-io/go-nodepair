package nodepair

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ipfs/go-log"
	"github.com/mudler/edgevpn/pkg/blockchain"
	"github.com/mudler/edgevpn/pkg/config"
	"github.com/mudler/edgevpn/pkg/services"

	"github.com/mudler/edgevpn/pkg/logger"
	node "github.com/mudler/edgevpn/pkg/node"
)

func newNode(cfg *PairConfig) (*node.Node, error) {
	llger := logger.New(log.LevelFatal)
	defaultInterval := 10 * time.Second

	loglevel := cfg.loglevel
	if loglevel == "" {
		loglevel = "fatal"
	}

	c := config.Config{
		Limit: config.ResourceLimit{
			Enable:   true,
			MaxConns: 100,
		},
		NetworkToken:   cfg.token,
		LowProfile:     false,
		LogLevel:       loglevel,
		Libp2pLogLevel: "fatal",
		Ledger: config.Ledger{
			SyncInterval:     defaultInterval,
			AnnounceInterval: defaultInterval,
		},
		NAT: config.NAT{
			Service:           true,
			Map:               true,
			RateLimit:         true,
			RateLimitGlobal:   10,
			RateLimitPeer:     10,
			RateLimitInterval: defaultInterval,
		},
		Discovery: config.Discovery{
			DHT:      true,
			MDNS:     true,
			Interval: 30 * time.Second,
		},
		Connection: config.Connection{
			HolePunch:      true,
			AutoRelay:      true,
			MaxConnections: 100,
		},
	}

	nodeOpts, _, err := c.ToOpts(llger)
	if err != nil {
		return nil, fmt.Errorf("parsing options: %w", err)
	}

	nodeOpts = append(nodeOpts, services.Alive(30*time.Second, 900*time.Second, 15*time.Minute)...)

	n, err := node.New(nodeOpts...)
	if err != nil {
		return nil, fmt.Errorf("creating a new node: %w", err)
	}
	return n, nil
}

// TokenReader is a function that reads a string and returns a token from it.
// A string can represent anything (uri, image file, etc.) which can be used to retrieve the connection token
type TokenReader func(string) string

// PairConfig is the pairing configuration structure
type PairConfig struct {
	tokenReader TokenReader
	token       string
	loglevel    string
}

// Apply sets the given option to the configuration
func (c *PairConfig) Apply(opts ...PairOption) error {
	for _, o := range opts {
		if err := o(c); err != nil {
			return err
		}
	}

	if c.tokenReader != nil {
		c.token = c.tokenReader(c.token)
	}

	if c.token == "" {
		return errors.New("no token supplied or couldn't read from providers (try with a better image or input source)")
	}

	return nil
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

// WithLogLevel sets the loglevel of the pairing operation
func WithLogLevel(t string) PairOption {
	return func(c *PairConfig) error {
		c.loglevel = t

		return nil
	}
}

// GenerateToken returns a token which can be used for pairing
func GenerateToken() string {
	d := node.GenerateNewConnectionData(int(^uint(0) >> 1))
	return d.Base64()
}

// Receive a payload during pairing
func Receive(ctx context.Context, payload interface{}, opts ...PairOption) error {
	c := &PairConfig{}

	if err := c.Apply(opts...); err != nil {
		return err
	}

	n, err := newNode(c)
	if err != nil {
		return err
	}

	if err := n.Start(ctx); err != nil {
		return err
	}

	l, err := n.Ledger()
	if err != nil {
		return err
	}

	l.AnnounceUpdate(ctx, 2*time.Second, "presence", n.Host().ID().String(), "")
	if _, err := waitNodes(ctx, l); err != nil {
		return err
	}

PAIRDATA:
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("waiting for the pairing payload: %w", ctx.Err())
		default:
			v, exists := l.GetKey("pairing", "data")
			if exists {
				v.Unmarshal(payload)
				l.AnnounceUpdate(ctx, 2*time.Second, "pairing", n.Host().ID().String(), "ok")
				break PAIRDATA
			}
			if err := sleep(ctx, 1*time.Second); err != nil {
				return fmt.Errorf("waiting for the pairing payload: %w", err)
			}
		}
	}

WAIT:
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("waiting for our own pairing acknowledgement: %w", ctx.Err())
		default:
			if _, exists := l.GetKey("pairing", n.Host().ID().String()); exists {
				break WAIT
			}
			if err := sleep(ctx, 1*time.Second); err != nil {
				return fmt.Errorf("waiting for our own pairing acknowledgement: %w", err)
			}
		}
	}

	return nil
}

func waitNodes(ctx context.Context, l *blockchain.Ledger) (active []string, err error) {
	enough := false
CHECK:
	for !enough {
		fmt.Println("Not enough nodes")
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("waiting for the other node to join the pairing network: %w", ctx.Err())
		default:
			nn := l.CurrentData()["presence"]
			active = []string{}
			for k := range nn {
				active = append(active, k)
			}
			enough = len(active) >= 2
			if enough {
				break CHECK
			} else if err := sleep(ctx, 10*time.Second); err != nil {
				return nil, fmt.Errorf("waiting for the other node to join the pairing network: %w", err)
			}
		}
	}

	return active, nil
}

// Send a payload during device pairing
func Send(ctx context.Context, payload interface{}, opts ...PairOption) error {
	c := &PairConfig{}

	if err := c.Apply(opts...); err != nil {
		return err
	}

	n, err := newNode(c)
	if err != nil {
		return err
	}

	// The error matters: nothing below fails when the node is not running, so
	// a discarded start error leaves Send announcing into a ledger no peer
	// gossips and then blocking in waitNodes until the caller's context ends.
	if err := n.Start(ctx); err != nil {
		return fmt.Errorf("starting node: %w", err)
	}

	l, err := n.Ledger()
	if err != nil {
		return err
	}

	l.AnnounceUpdate(ctx, 3*time.Second, "pairing", "data", payload)
	l.AnnounceUpdate(ctx, 3*time.Second, "presence", n.Host().ID().String(), "")
	active, err := waitNodes(ctx, l)
	if err != nil {
		return err
	}

PAIRING:
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("waiting for the other node to accept the payload: %w", ctx.Err())
		default:
			d := l.CurrentData()
			fmt.Println("Pairing in progress")
			fmt.Println(d)
			for _, a := range active {
				fmt.Println("Active", a)

				if n.Host().ID().String() == a {
					continue
				}
				_, exists := l.GetKey("pairing", a)
				if exists {
					break PAIRING
				}
				if err := sleep(ctx, 1*time.Second); err != nil {
					return fmt.Errorf("waiting for the other node to accept the payload: %w", err)
				}
			}
		}
	}
	return nil
}

// sleep waits for d, or returns early with the context's error when the
// context ends first. A bare time.Sleep in a polling loop makes the loop
// notice a finished context up to d late, so a caller's deadline is only
// honoured to within the poll interval.
func sleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
