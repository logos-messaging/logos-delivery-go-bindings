package kernel

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/logos-messaging/logos-delivery-go-bindings/pkg/kernel/common"
)

// requestTimeout is the deadline applied to a context that carries none of its
// own. It has to be a real number: the library passes the value straight to
// chronos' withTimeout, where zero milliseconds expires immediately.
const requestTimeout = 30 * time.Second

// timeoutMillis renders a context's remaining time as the millisecond timeout
// the library expects, falling back to def when the context has no deadline.
func timeoutMillis(ctx context.Context, def time.Duration) int {
	deadline, ok := ctx.Deadline()
	if !ok {
		return int(def.Milliseconds())
	}

	remaining := time.Until(deadline)
	if remaining <= 0 {
		return 0
	}
	return int(remaining.Milliseconds())
}

// FormatWakuRelayTopic renders the pubsub topic for a cluster's shard.
func FormatWakuRelayTopic(clusterID uint16, shard uint16) string {
	return fmt.Sprintf("/waku/2/rs/%d/%d", clusterID, shard)
}

// GetFreePortIfNeeded replaces a zero TCP or UDP port with one the OS reports
// as free, leaving non-zero ports untouched.
func GetFreePortIfNeeded(tcpPort int, discV5UDPPort int) (int, int, error) {
	if tcpPort == 0 {
		for i := 0; i < 10; i++ {
			tcpAddr, err := net.ResolveTCPAddr("tcp", net.JoinHostPort("localhost", "0"))
			if err != nil {
				Warn("unable to resolve tcp addr: %v", err)
				continue
			}
			tcpListener, err := net.ListenTCP("tcp", tcpAddr)
			if err != nil {
				Warn("unable to listen on addr: addr=%v, error=%v", tcpAddr, err)
				continue
			}
			tcpPort = tcpListener.Addr().(*net.TCPAddr).Port
			_ = tcpListener.Close()
			break
		}
		if tcpPort == 0 {
			return -1, -1, errors.New("could not obtain a free TCP port")
		}
	}

	if discV5UDPPort == 0 {
		for i := 0; i < 10; i++ {
			udpAddr, err := net.ResolveUDPAddr("udp", net.JoinHostPort("localhost", "0"))
			if err != nil {
				Warn("unable to resolve udp addr: %v", err)
				continue
			}
			udpListener, err := net.ListenUDP("udp", udpAddr)
			if err != nil {
				Warn("unable to listen on addr: addr=%v, error=%v", udpAddr, err)
				continue
			}
			discV5UDPPort = udpListener.LocalAddr().(*net.UDPAddr).Port
			_ = udpListener.Close()
			break
		}
		if discV5UDPPort == 0 {
			return -1, -1, errors.New("could not obtain a free UDP port")
		}
	}

	return tcpPort, discV5UDPPort, nil
}

// StartWakuNode creates a node from a legacy flat configuration and starts it,
// filling in free TCP and DiscV5 UDP ports where the configuration leaves them
// at zero. A nil configuration uses DefaultWakuConfig.
func StartWakuNode(nodeName string, customCfg *common.WakuConfig) (*Node, error) {
	nodeCfg := DefaultWakuConfig
	if customCfg != nil {
		nodeCfg = *customCfg
	}

	tcpPort, udpPort, err := GetFreePortIfNeeded(nodeCfg.TcpPort, nodeCfg.Discv5UdpPort)
	if err != nil {
		Error("Failed to allocate unique ports: %v", err)
		tcpPort, udpPort = 0, 0
	}
	if nodeCfg.TcpPort == 0 {
		nodeCfg.TcpPort = tcpPort
	}
	if nodeCfg.Discv5UdpPort == 0 {
		nodeCfg.Discv5UdpPort = udpPort
	}

	node, err := NewFromWakuConfig(&nodeCfg, nodeName)
	if err != nil {
		return nil, err
	}

	if err := node.Start(); err != nil {
		_ = node.Close()
		return nil, err
	}
	return node, nil
}
