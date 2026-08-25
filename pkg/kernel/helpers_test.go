package kernel

import (
	"fmt"
	"net"
	"testing"

	"github.com/logos-messaging/logos-delivery-go-bindings/pkg/kernel/common"
)

// requiresNode marks a test that starts a real node and talks to a network, so
// that `go test -short` runs only the tests needing neither.
func requiresNode(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping: needs a live node")
	}
}

// freePort reserves a port the OS reports as free on the given network.
//
// Tests allocate ports up front instead of passing 0, because the library does
// not treat a zero DiscV5 UDP port as "pick one": every node would try the same
// default and all but the first would fail to bind.
func freePort(network string) (int, error) {
	switch network {
	case "tcp":
		addr, err := net.ResolveTCPAddr(network, net.JoinHostPort("localhost", "0"))
		if err != nil {
			return 0, err
		}
		listener, err := net.ListenTCP(network, addr)
		if err != nil {
			return 0, err
		}
		defer func() { _ = listener.Close() }()
		return listener.Addr().(*net.TCPAddr).Port, nil

	case "udp":
		addr, err := net.ResolveUDPAddr(network, net.JoinHostPort("localhost", "0"))
		if err != nil {
			return 0, err
		}
		listener, err := net.ListenUDP(network, addr)
		if err != nil {
			return 0, err
		}
		defer func() { _ = listener.Close() }()
		return listener.LocalAddr().(*net.UDPAddr).Port, nil
	}
	return 0, fmt.Errorf("unsupported network %q", network)
}

func freeTCPPort(t *testing.T) int {
	t.Helper()
	port, err := freePort("tcp")
	if err != nil {
		t.Fatalf("free tcp port: %v", err)
	}
	return port
}

func freeUDPPort(t *testing.T) int {
	t.Helper()
	port, err := freePort("udp")
	if err != nil {
		t.Fatalf("free udp port: %v", err)
	}
	return port
}

// StartWakuNode creates a node from a legacy flat configuration and starts it,
// filling in free ports where the configuration leaves them at zero. A nil
// configuration uses DefaultWakuConfig.
func StartWakuNode(customCfg *common.WakuConfig) (*Node, error) {
	nodeCfg := DefaultWakuConfig
	if customCfg != nil {
		nodeCfg = *customCfg
	}

	if nodeCfg.TcpPort == 0 {
		port, err := freePort("tcp")
		if err != nil {
			return nil, err
		}
		nodeCfg.TcpPort = port
	}
	if nodeCfg.Discv5UdpPort == 0 {
		port, err := freePort("udp")
		if err != nil {
			return nil, err
		}
		nodeCfg.Discv5UdpPort = port
	}

	node, err := NewFromWakuConfig(&nodeCfg)
	if err != nil {
		return nil, err
	}

	if err := node.Start(); err != nil {
		_ = node.Close()
		return nil, err
	}
	return node, nil
}
