package kernel

import (
	"context"
	"fmt"
	"strconv"

	"github.com/multiformats/go-multiaddr"

	"github.com/logos-messaging/logos-delivery-go-bindings/internal/ffi"
)

// Discovery is a Node's peer discovery surface: DiscV5, DNS discovery and peer
// exchange. Take one with Node.Discovery.
type Discovery struct{ n *Node }

// StartDiscV5 starts DiscV5 peer discovery.
func (d Discovery) StartDiscV5() error {
	if err := d.n.check(); err != nil {
		return err
	}

	if err := ffi.StartDiscV5(d.n.h); err != nil {
		Error("Failed to start DiscV5 for %s: %v", d.n.name, err)
		return fmt.Errorf("kernel: start discv5: %w", err)
	}

	Debug("Successfully started DiscV5 for %s", d.n.name)
	return nil
}

// StopDiscV5 stops DiscV5 peer discovery.
func (d Discovery) StopDiscV5() error {
	if err := d.n.check(); err != nil {
		return err
	}

	if err := ffi.StopDiscV5(d.n.h); err != nil {
		Error("Failed to stop DiscV5 for %s: %v", d.n.name, err)
		return fmt.Errorf("kernel: stop discv5: %w", err)
	}

	Debug("Successfully stopped DiscV5 for %s", d.n.name)
	return nil
}

// DNSDiscovery resolves an ENR tree URL and returns the multiaddresses it
// advertises. A ctx without a deadline gets the package default of 30s.
func (d Discovery) DNSDiscovery(
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

// PeerExchangeRequest asks peer exchange for numPeers peers and returns how
// many were received.
func (d Discovery) PeerExchangeRequest(numPeers uint64) (uint64, error) {
	if err := d.n.check(); err != nil {
		return 0, err
	}

	countStr, err := ffi.PeerExchangeRequest(d.n.h, numPeers)
	if err != nil {
		return 0, fmt.Errorf("kernel: peer exchange request: %w", err)
	}
	return strconv.ParseUint(countStr, 10, 64)
}
