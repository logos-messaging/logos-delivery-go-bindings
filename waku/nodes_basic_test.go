package waku

import (
	"testing"
	"time"

	"github.com/logos-messaging/logos-delivery-go-bindings/waku/common"
	"github.com/stretchr/testify/require"
)

func TestBasicWakuNodes(t *testing.T) {
	Debug("Starting TestBasicWakuNodes")

	nodeCfg := DefaultWakuConfig
	nodeCfg.Relay = true

	Debug("Starting the WakuNode")
	node, err := StartWakuNode("node", &nodeCfg)
	require.NoError(t, err, "Failed to create the WakuNode")

	// Use defer to ensure proper cleanup
	defer func() {
		node.StopAndDestroy()
	}()

	Debug("Successfully created the WakuNode")
	time.Sleep(2 * time.Second)

	Debug("TestBasicWakuNodes completed successfully")
}

/* artifact https://github.com/logos-messaging/logos-delivery-go-bindings/issues/40 */
func TestNodeRestart(t *testing.T) {
	t.Skip("Skipping test for open artifact ")
	Debug("Starting TestNodeRestart")

	Debug("Creating Node")
	nodeConfig := DefaultWakuConfig
	node, err := StartWakuNode("TestNode", &nodeConfig)
	require.NoError(t, err, "Failed to start Waku node")
	defer node.StopAndDestroy()

	Debug("Node started successfully")

	Debug("Fetching ENR before stopping the node")
	enrBefore, err := node.ENR()
	require.NoError(t, err, "Failed to get ENR before stopping")
	require.NotEmpty(t, enrBefore, "ENR should not be empty before stopping")
	Debug("ENR before stopping: %s", enrBefore)

	Debug("Stopping the Node")
	err = node.Stop()
	require.NoError(t, err, "Failed to stop Waku node")
	Debug("Node stopped successfully")

	Debug("Restarting the Node")
	err = node.Start()
	require.NoError(t, err, "Failed to restart Waku node")
	Debug("Node restarted successfully")

	Debug("Fetching ENR after restarting the node")
	enrAfter, err := node.ENR()
	require.NoError(t, err, "Failed to get ENR after restarting")
	require.NotEmpty(t, enrAfter, "ENR should not be empty after restart")
	Debug("ENR after restarting: %s", enrAfter)

	Debug("Comparing ENRs before and after restart")
	require.Equal(t, enrBefore, enrAfter, "ENR should remain the same after node restart")

	Debug("TestNodeRestart completed successfully")
}
func TestDoubleStart(t *testing.T) {

	tcpPort, udpPort, err := GetFreePortIfNeeded(0, 0)
	require.NoError(t, err)

	config := common.WakuConfig{
		Relay:           true,
		Store:           true,
		LogLevel:        "DEBUG",
		Discv5Discovery: true,
		ClusterID:       16,
		Shards:          []uint16{64},
		Discv5UdpPort:   udpPort,
		TcpPort:         tcpPort,
	}

	node, err := NewWakuNode(&config, "node")
	require.NoError(t, err)
	defer node.StopAndDestroy()

	// start node
	require.NoError(t, node.Start())
	// now attempt to start again
	require.NoError(t, node.Start())

}

func TestDoubleStop(t *testing.T) {

	tcpPort, udpPort, err := GetFreePortIfNeeded(0, 0)
	require.NoError(t, err)

	config := common.WakuConfig{
		Relay:           true,
		Store:           true,
		LogLevel:        "DEBUG",
		Discv5Discovery: true,
		ClusterID:       16,
		Shards:          []uint16{64},
		Discv5UdpPort:   udpPort,
		TcpPort:         tcpPort,
	}

	node, err := NewWakuNode(&config, "node")
	require.NoError(t, err)
	defer node.StopAndDestroy()

	// start node
	require.NoError(t, node.Start())

	// stop node
	require.NoError(t, node.Stop())
	// now attempt to stop it again
	require.NoError(t, node.Stop())

}
