package kernel

import (
	"testing"
	"time"

	"github.com/logos-messaging/logos-delivery-go-bindings/pkg/kernel/common"
	"github.com/stretchr/testify/require"
)

func TestBasicWakuNodes(t *testing.T) {
	requiresNode(t)
	logDebug("Starting TestBasicWakuNodes")

	nodeCfg := DefaultWakuConfig
	nodeCfg.Relay = true

	logDebug("Starting the WakuNode")
	node, err := StartWakuNode(&nodeCfg)
	require.NoError(t, err, "Failed to create the WakuNode")

	// Use defer to ensure proper cleanup
	defer func() {
		_ = node.Close()
	}()

	logDebug("Successfully created the WakuNode")
	time.Sleep(2 * time.Second)

	logDebug("TestBasicWakuNodes completed successfully")
}

/* artifact https://github.com/logos-messaging/logos-delivery-go-bindings/issues/40 */
func TestNodeRestart(t *testing.T) {
	requiresNode(t)
	t.Skip("Skipping test for open artifact ")
	logDebug("Starting TestNodeRestart")

	logDebug("Creating Node")
	nodeConfig := DefaultWakuConfig
	node, err := StartWakuNode(&nodeConfig)
	require.NoError(t, err, "Failed to start Waku node")
	defer func() { _ = node.Close() }()

	logDebug("Node started successfully")

	logDebug("Fetching ENR before stopping the node")
	enrBefore, err := node.Debug().ENR()
	require.NoError(t, err, "Failed to get ENR before stopping")
	require.NotEmpty(t, enrBefore, "ENR should not be empty before stopping")
	logDebug("ENR before stopping: %s", enrBefore)

	logDebug("Stopping the Node")
	err = node.Stop()
	require.NoError(t, err, "Failed to stop Waku node")
	logDebug("Node stopped successfully")

	logDebug("Restarting the Node")
	err = node.Start()
	require.NoError(t, err, "Failed to restart Waku node")
	logDebug("Node restarted successfully")

	logDebug("Fetching ENR after restarting the node")
	enrAfter, err := node.Debug().ENR()
	require.NoError(t, err, "Failed to get ENR after restarting")
	require.NotEmpty(t, enrAfter, "ENR should not be empty after restart")
	logDebug("ENR after restarting: %s", enrAfter)

	logDebug("Comparing ENRs before and after restart")
	require.Equal(t, enrBefore, enrAfter, "ENR should remain the same after node restart")

	logDebug("TestNodeRestart completed successfully")
}
func TestDoubleStart(t *testing.T) {
	requiresNode(t)

	config := common.WakuConfig{
		Relay:           true,
		Store:           true,
		LogLevel:        "DEBUG",
		Discv5Discovery: true,
		ClusterID:       16,
		Shards:          []uint16{64},
		Discv5UdpPort:   freeUDPPort(t),
		TcpPort:         freeTCPPort(t),
	}

	node, err := NewFromWakuConfig(&config)
	require.NoError(t, err)
	defer func() { _ = node.Close() }()

	// start node
	require.NoError(t, node.Start())
	// now attempt to start again
	require.NoError(t, node.Start())

}

func TestDoubleStop(t *testing.T) {
	requiresNode(t)

	config := common.WakuConfig{
		Relay:           true,
		Store:           true,
		LogLevel:        "DEBUG",
		Discv5Discovery: true,
		ClusterID:       16,
		Shards:          []uint16{64},
		Discv5UdpPort:   freeUDPPort(t),
		TcpPort:         freeTCPPort(t),
	}

	node, err := NewFromWakuConfig(&config)
	require.NoError(t, err)
	defer func() { _ = node.Close() }()

	// start node
	require.NoError(t, node.Start())

	// stop node
	require.NoError(t, node.Stop())
	// now attempt to stop it again
	require.NoError(t, node.Stop())

}
