package kernel

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	libp2pproto "github.com/libp2p/go-libp2p/core/protocol"
	"github.com/multiformats/go-multiaddr"

	"github.com/logos-messaging/logos-delivery-go-bindings/internal/ffi"
	"github.com/logos-messaging/logos-delivery-go-bindings/pkg/kernel/common"
	"github.com/logos-messaging/logos-delivery-go-bindings/pkg/kernel/utils"
)

// Peers is a Node's peer management surface: dialling, disconnecting and
// inspecting the peer store. Take one with Node.Peers.
type Peers struct {
	n *Node
}

// Connect dials a peer multiaddress. A ctx without a deadline gets the package
// default of 30s.
func (p *Peers) Connect(ctx context.Context, addr multiaddr.Multiaddr) error {
	if err := p.n.check(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	if err := ffi.Connect(p.n.h, addr.String(), timeoutMillis(ctx, requestTimeout)); err != nil {
		return fmt.Errorf("kernel: connect: %w", err)
	}
	return nil
}

// ConnectTo dials another node by its first listen address.
func (p *Peers) ConnectTo(ctx context.Context, target *Node) error {
	if target == nil {
		return errors.New("kernel: connect: target node is nil")
	}

	addrs, err := target.Debug().ListenAddresses()
	if err != nil {
		return err
	}
	if len(addrs) == 0 {
		return errors.New("kernel: connect: target node has no listen addresses")
	}

	return p.Connect(ctx, addrs[0])
}

// Dial dials a peer multiaddress over a specific protocol. A ctx without a
// deadline gets the package default of 30s.
func (p *Peers) Dial(ctx context.Context, addr multiaddr.Multiaddr, protocol libp2pproto.ID) error {
	if err := p.n.check(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	if err := ffi.DialPeer(
		p.n.h, addr.String(), string(protocol), timeoutMillis(ctx, requestTimeout),
	); err != nil {
		return fmt.Errorf("kernel: dial peer: %w", err)
	}
	return nil
}

// DialByID dials a known peer over a specific protocol. A ctx without a
// deadline gets the package default of 30s.
func (p *Peers) DialByID(ctx context.Context, id peer.ID, protocol libp2pproto.ID) error {
	if err := p.n.check(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	if err := ffi.DialPeerByID(
		p.n.h, id.String(), string(protocol), timeoutMillis(ctx, requestTimeout),
	); err != nil {
		return fmt.Errorf("kernel: dial peer by id: %w", err)
	}
	return nil
}

// Disconnect drops the connection to a peer.
func (p *Peers) Disconnect(id peer.ID) error {
	if err := p.n.check(); err != nil {
		return err
	}

	if err := ffi.DisconnectPeerByID(p.n.h, id.String()); err != nil {
		return fmt.Errorf("kernel: disconnect peer: %w", err)
	}
	return nil
}

// DisconnectFrom drops the connection to another node.
func (p *Peers) DisconnectFrom(target *Node) error {
	if target == nil {
		return errors.New("kernel: disconnect: target node is nil")
	}

	id, err := target.Debug().PeerID()
	if err != nil {
		return err
	}

	return p.Disconnect(id)
}

// DisconnectAll drops every peer connection.
func (p *Peers) DisconnectAll() error {
	if err := p.n.check(); err != nil {
		return err
	}

	if err := ffi.DisconnectAllPeers(p.n.h); err != nil {
		return fmt.Errorf("kernel: disconnect all peers: %w", err)
	}
	return nil
}

// Connected returns the currently connected peers.
func (p *Peers) Connected() (peer.IDSlice, error) {
	if err := p.n.check(); err != nil {
		return nil, err
	}

	list, err := ffi.GetConnectedPeers(p.n.h)
	if err != nil {
		return nil, fmt.Errorf("kernel: connected peers: %w", err)
	}
	return parsePeerIDs(list)
}

// NumConnected returns the number of currently connected peers.
func (p *Peers) NumConnected() (int, error) {
	peers, err := p.Connected()
	if err != nil {
		return 0, err
	}
	return len(peers), nil
}

// ConnectedInfo returns the protocols and addresses of every connected peer.
func (p *Peers) ConnectedInfo() (common.PeersData, error) {
	if err := p.n.check(); err != nil {
		return nil, err
	}

	jsonStr, err := ffi.GetConnectedPeersInfo(p.n.h)
	if err != nil {
		return nil, fmt.Errorf("kernel: connected peers info: %w", err)
	}
	if jsonStr == "" {
		return nil, nil
	}

	data, err := common.ParsePeerInfoFromJSON(jsonStr)
	if err != nil {
		return nil, fmt.Errorf("kernel: parse connected peers info: %w", err)
	}
	return data, nil
}

// FromPeerStore returns every peer the peer store knows about, connected or
// not.
func (p *Peers) FromPeerStore() (peer.IDSlice, error) {
	if err := p.n.check(); err != nil {
		return nil, err
	}

	list, err := ffi.GetPeerIDsFromPeerStore(p.n.h)
	if err != nil {
		return nil, fmt.Errorf("kernel: peer store: %w", err)
	}
	return parsePeerIDs(list)
}

// ByProtocol returns the known peers that support a protocol.
func (p *Peers) ByProtocol(protocol libp2pproto.ID) (peer.IDSlice, error) {
	if err := p.n.check(); err != nil {
		return nil, err
	}

	list, err := ffi.GetPeerIDsByProtocol(p.n.h, string(protocol))
	if err != nil {
		return nil, fmt.Errorf("kernel: peers by protocol: %w", err)
	}
	return parsePeerIDs(list)
}

// Ping measures the round-trip time to a peer. A ctx without a deadline gets
// the package default of 30s.
func (p *Peers) Ping(ctx context.Context, peerInfo peer.AddrInfo) (time.Duration, error) {
	if err := p.n.check(); err != nil {
		return 0, err
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	addr, err := peerAddr(peerInfo)
	if err != nil {
		return 0, err
	}

	rttStr, err := ffi.PingPeer(p.n.h, addr, timeoutMillis(ctx, requestTimeout))
	if err != nil {
		return 0, fmt.Errorf("kernel: ping peer: %w", err)
	}

	rtt, err := strconv.ParseInt(rttStr, 10, 64)
	if err != nil {
		return 0, err
	}
	return time.Duration(rtt), nil
}

// peerAddr renders a peer's first address as the peer-id-encapsulated
// multiaddress string the library expects. Only one is sent: the entry points
// parse their argument as a single multiaddress, so a joined list of them
// fails to parse.
func peerAddr(peerInfo peer.AddrInfo) (string, error) {
	encapsulated := utils.EncapsulatePeerID(peerInfo.ID, peerInfo.Addrs...)
	if len(encapsulated) == 0 {
		return "", fmt.Errorf("kernel: peer %s has no addresses", peerInfo.ID)
	}
	return encapsulated[0].String(), nil
}
