package kernel

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"

	"github.com/logos-messaging/logos-delivery-go-bindings/internal/ffi"
)

// Debug is a Node's own identity and health surface. Take one with Node.Debug.
type Debug struct {
	n *Node
}

// PeerID returns the node's own peer id.
func (d *Debug) PeerID() (peer.ID, error) {
	if err := d.n.check(); err != nil {
		return "", err
	}

	idStr, err := ffi.GetMyPeerID(d.n.h)
	if err != nil {
		return "", fmt.Errorf("kernel: peer id: %w", err)
	}

	id, err := peer.Decode(idStr)
	if err != nil {
		return "", fmt.Errorf("kernel: decode peer id: %w", err)
	}
	return id, nil
}

// ListenAddresses returns the multiaddresses the node listens on.
func (d *Debug) ListenAddresses() ([]multiaddr.Multiaddr, error) {
	if err := d.n.check(); err != nil {
		return nil, err
	}

	addrs, err := ffi.ListenAddresses(d.n.h)
	if err != nil {
		return nil, fmt.Errorf("kernel: listen addresses: %w", err)
	}
	return parseMultiaddrs(addrs)
}

// ENR returns the node's own ENR record.
func (d *Debug) ENR() (*enode.Node, error) {
	if err := d.n.check(); err != nil {
		return nil, err
	}

	enrStr, err := ffi.GetMyENR(d.n.h)
	if err != nil {
		return nil, fmt.Errorf("kernel: enr: %w", err)
	}

	record, err := enode.Parse(enode.ValidSchemes, enrStr)
	if err != nil {
		return nil, fmt.Errorf("kernel: parse enr: %w", err)
	}
	return record, nil
}

// Version returns the library version the node runs.
func (d *Debug) Version() (string, error) {
	if err := d.n.check(); err != nil {
		return "", err
	}

	version, err := ffi.Version(d.n.h)
	if err != nil {
		return "", fmt.Errorf("kernel: version: %w", err)
	}
	return version, nil
}

// IsOnline reports whether the node considers itself connected to the network.
func (d *Debug) IsOnline() (bool, error) {
	if err := d.n.check(); err != nil {
		return false, err
	}

	online, err := ffi.IsOnline(d.n.h)
	if err != nil {
		return false, fmt.Errorf("kernel: is online: %w", err)
	}
	return online == "true", nil
}

// Metrics returns the node's metrics in Prometheus text format.
func (d *Debug) Metrics() (string, error) {
	if err := d.n.check(); err != nil {
		return "", err
	}

	metrics, err := ffi.GetMetrics(d.n.h)
	if err != nil {
		return "", fmt.Errorf("kernel: metrics: %w", err)
	}
	if metrics == "" {
		return "", errors.New("kernel: metrics: empty response")
	}
	return metrics, nil
}
