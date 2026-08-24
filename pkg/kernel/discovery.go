package kernel

import (
	"context"
	"fmt"
	"strconv"

	"github.com/multiformats/go-multiaddr"

	"github.com/logos-messaging/logos-delivery-go-bindings/internal/ffi"
)

// DiscV5 is a Node's DiscV5 peer discovery surface. Take one with Node.DiscV5.
type DiscV5 struct{ n *Node }

// Start starts DiscV5 peer discovery.
func (d *DiscV5) Start() error {
	if err := d.n.check(); err != nil {
		return err
	}

	if err := ffi.StartDiscV5(d.n.h); err != nil {
		return fmt.Errorf("kernel: start discv5: %w", err)
	}
	return nil
}

// Stop stops DiscV5 peer discovery.
func (d *DiscV5) Stop() error {
	if err := d.n.check(); err != nil {
		return err
	}

	if err := ffi.StopDiscV5(d.n.h); err != nil {
		return fmt.Errorf("kernel: stop discv5: %w", err)
	}
	return nil
}

// PeerExchange is a Node's peer exchange protocol surface. Take one with
// Node.PeerExchange.
type PeerExchange struct{ n *Node }

// Request asks peer exchange for numPeers peers and returns how many were
// received.
func (p *PeerExchange) Request(numPeers uint64) (uint64, error) {
	if err := p.n.check(); err != nil {
		return 0, err
	}

	countStr, err := ffi.PeerExchangeRequest(p.n.h, numPeers)
	if err != nil {
		return 0, fmt.Errorf("kernel: peer exchange request: %w", err)
	}
	return strconv.ParseUint(countStr, 10, 64)
}

// DNSDiscovery is a Node's DNS-based peer discovery surface. Take one with
// Node.DNSDiscovery.
type DNSDiscovery struct{ n *Node }

// Resolve resolves an ENR tree URL and returns the multiaddresses it
// advertises. A ctx without a deadline gets the package default of 30s.
func (d *DNSDiscovery) Resolve(
	ctx context.Context, enrTreeURL, nameDNSServer string,
) ([]multiaddr.Multiaddr, error) {
	if err := d.n.check(); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	list, err := ffi.DnsDiscovery(
		d.n.h, enrTreeURL, nameDNSServer, timeoutMillis(ctx, requestTimeout),
	)
	if err != nil {
		return nil, fmt.Errorf("kernel: dns discovery: %w", err)
	}
	return parseMultiaddrs(list)
}
