package kernel

import (
	"slices"
	"testing"
	"time"

	"github.com/logos-messaging/logos-delivery-go-bindings/pkg/kernel/common"
	"github.com/stretchr/testify/require"
)

// Test node connect & disconnect peers
func TestDisconnectPeerNodes(t *testing.T) {
	Debug("Starting TestDisconnectPeerNodes")

	nodeA, err := StartWakuNode("nodeA", nil)
	require.NoError(t, err, "Failed to start Node A")
	defer nodeA.StopAndDestroy()

	nodeB, err := StartWakuNode("nodeB", nil)
	require.NoError(t, err, "Failed to start Node B")
	defer nodeB.StopAndDestroy()

	Debug("Connecting Node A to Node B")
	err = nodeA.ConnectPeer(nodeB)
	require.NoError(t, err, "Failed to connect nodes")

	Debug("Verifying connection between Node A and Node B")
	connectedPeers, err := nodeA.GetConnectedPeers()
	require.NoError(t, err, "Failed to get connected peers for Node A")
	nodeBPeerID, err := nodeB.PeerID()
	require.NoError(t, err, "Failed to get PeerID for Node B")
	require.True(t, slices.Contains(connectedPeers, nodeBPeerID), "Node B should be a peer of Node A before disconnection")

	Debug("Disconnecting Node A from Node B")
	err = nodeA.DisconnectPeer(nodeB)
	require.NoError(t, err, "Failed to disconnect nodes")

	Debug("Verifying disconnection between Node A and Node B")
	connectedPeers, err = nodeA.GetConnectedPeers()
	require.NoError(t, err, "Failed to get connected peers for Node A after disconnection")
	require.False(t, slices.Contains(connectedPeers, nodeBPeerID), "Node B should no longer be a peer of Node A after disconnection")
	Debug("Test completed successfully: Node B was disconnected from Node A")
}

func TestConnectMultipleNodesToSingleNode(t *testing.T) {
	Debug("Starting TestConnectMultipleNodesToSingleNode")

	Debug("Creating 3 nodes with automatically assigned ports")

	node1, err := StartWakuNode("node1", nil)
	require.NoError(t, err, "Failed to start Node 1")
	defer func() {
		Debug("Stopping and destroying Node 1")
		node1.StopAndDestroy()
	}()

	node2, err := StartWakuNode("node2", nil)
	require.NoError(t, err, "Failed to start Node 2")
	defer func() {
		Debug("Stopping and destroying Node 2")
		node2.StopAndDestroy()
	}()

	node3, err := StartWakuNode("node3", nil)
	require.NoError(t, err, "Failed to start Node 3")
	defer func() {
		Debug("Stopping and destroying Node 3")
		node3.StopAndDestroy()
	}()

	Debug("Connecting Node 2 to Node 1")
	err = node2.ConnectPeer(node1)
	require.NoError(t, err, "Failed to connect Node 2 to Node 1")

	Debug("Connecting Node 3 to Node 1")
	err = node3.ConnectPeer(node1)
	require.NoError(t, err, "Failed to connect Node 3 to Node 1")

	Debug("Verifying connected peers for Node 1")
	connectedPeers, err := node1.GetConnectedPeers()
	require.NoError(t, err, "Failed to get connected peers for Node 1")
	node3PeerID, err := node3.PeerID()
	require.NoError(t, err, "Failed to get PeerID for Node 1")
	node2PeerID, err := node2.PeerID()
	require.NoError(t, err, "Failed to get PeerID for Node 2")

	require.True(t, slices.Contains(connectedPeers, node3PeerID), "Node 3 should be a peer of Node 1")
	require.True(t, slices.Contains(connectedPeers, node2PeerID), "Node 2 should be a peer of Node 1")

	Debug("Test completed successfully: multiple nodes connected to a single node and verified peers")
}

func TestConnectUsingMultipleStaticPeers(t *testing.T) {
	Debug("Starting TestConnectUsingMultipleStaticPeers")

	node1, err := StartWakuNode("node1", nil)
	require.NoError(t, err, "Failed to start Node 1")

	node2, err := StartWakuNode("node2", nil)
	require.NoError(t, err, "Failed to start Node 2")

	node3, err := StartWakuNode("node3", nil)
	require.NoError(t, err, "Failed to start Node 3")

	addr1, err := node1.ListenAddresses()
	require.NoError(t, err, "Failed to get listen addresses for Node 1")

	addr2, err := node2.ListenAddresses()
	require.NoError(t, err, "Failed to get listen addresses for Node 2")

	addr3, err := node3.ListenAddresses()
	require.NoError(t, err, "Failed to get listen addresses for Node 3")

	node4Config := DefaultWakuConfig
	node4Config.Discv5Discovery = false
	node4Config.Staticnodes = []string{addr1[0].String(), addr2[0].String(), addr3[0].String()}

	node4, err := StartWakuNode("node4", &node4Config)
	require.NoError(t, err, "Failed to start Node 4")

	defer func() {
		Debug("Stopping and destroying all Waku nodes")
		node1.StopAndDestroy()
		node2.StopAndDestroy()
		node3.StopAndDestroy()
		node4.StopAndDestroy()
	}()

	Debug("Verifying connected peers for Node 4")
	connectedPeers, err := node4.GetConnectedPeers()
	require.NoError(t, err, "Failed to get connected peers for Node 4")

	node1PeerID, err := node1.PeerID()
	require.NoError(t, err, "Failed to get PeerID for Node 1")
	node2PeerID, err := node2.PeerID()
	require.NoError(t, err, "Failed to get PeerID for Node 2")
	node3PeerID, err := node3.PeerID()
	require.NoError(t, err, "Failed to get PeerID for Node 3")

	require.True(t, slices.Contains(connectedPeers, node1PeerID), "Node 1 should be a peer of Node 4")
	require.True(t, slices.Contains(connectedPeers, node2PeerID), "Node 2 should be a peer of Node 4")
	require.True(t, slices.Contains(connectedPeers, node3PeerID), "Node 3 should be a peer of Node 4")

	Debug("Test passed: multiple nodes connected to a single node using Static Peers")
}

func TestConnectedPeersInfo(t *testing.T) {
	Debug("Starting TestPeerData")

	node1, err := StartWakuNode("node1", nil)
	require.NoError(t, err, "Failed to start Node 1")

	node2, err := StartWakuNode("node2", nil)
	require.NoError(t, err, "Failed to start Node 2")

	node3, err := StartWakuNode("node3", nil)
	require.NoError(t, err, "Failed to start Node 3")

	addr1, err := node1.ListenAddresses()
	require.NoError(t, err, "Failed to get listen addresses for Node 1")

	addr2, err := node2.ListenAddresses()
	require.NoError(t, err, "Failed to get listen addresses for Node 2")

	addr3, err := node3.ListenAddresses()
	require.NoError(t, err, "Failed to get listen addresses for Node 3")

	node4Config := DefaultWakuConfig
	node4Config.Discv5Discovery = false
	node4Config.Staticnodes = []string{addr1[0].String(), addr2[0].String(), addr3[0].String()}

	node4, err := StartWakuNode("node4", &node4Config)
	require.NoError(t, err, "Failed to start Node 4")

	defer func() {
		Debug("Stopping and destroying all Waku nodes")
		node1.StopAndDestroy()
		node2.StopAndDestroy()
		node3.StopAndDestroy()
		node4.StopAndDestroy()
	}()

	Debug("Verifying connected peers for Node 4")
	connectedPeers, err := node4.GetConnectedPeers()
	require.NoError(t, err, "Failed to get connected peers for Node 4")

	node1PeerID, err := node1.PeerID()
	require.NoError(t, err, "Failed to get PeerID for Node 1")
	node2PeerID, err := node2.PeerID()
	require.NoError(t, err, "Failed to get PeerID for Node 2")
	node3PeerID, err := node3.PeerID()
	require.NoError(t, err, "Failed to get PeerID for Node 3")

	require.True(t, slices.Contains(connectedPeers, node1PeerID), "Node 1 should be a peer of Node 4")
	require.True(t, slices.Contains(connectedPeers, node2PeerID), "Node 2 should be a peer of Node 4")
	require.True(t, slices.Contains(connectedPeers, node3PeerID), "Node 3 should be a peer of Node 4")

	time.Sleep(1 * time.Second)

	peersInfo, err := node4.GetConnectedPeersInfo()
	require.NoError(t, err, "Failed to get node 4's connected peers info")

	require.Equal(t, 3, len(peersInfo), "Expected Node 4's connected peers info to have 3 entries")

	// Verify the addresses contain the expected peer IDs when encapsulated
	node1DerivedAddr := common.EncapsulatePeerID(node1PeerID, peersInfo[node1PeerID].Addresses...)
	require.NotEmpty(t, node1DerivedAddr, "Node1's derived address should not be empty")
	require.Contains(t, node1DerivedAddr[0].String(), node1PeerID.String(), "Node1's derived address should contain its peer ID")

	node2DerivedAddr := common.EncapsulatePeerID(node2PeerID, peersInfo[node2PeerID].Addresses...)
	require.NotEmpty(t, node2DerivedAddr, "Node2's derived address should not be empty")
	require.Contains(t, node2DerivedAddr[0].String(), node2PeerID.String(), "Node2's derived address should contain its peer ID")

	node3DerivedAddr := common.EncapsulatePeerID(node3PeerID, peersInfo[node3PeerID].Addresses...)
	require.NotEmpty(t, node3DerivedAddr, "Node3's derived address should not be empty")
	require.Contains(t, node3DerivedAddr[0].String(), node3PeerID.String(), "Node3's derived address should contain its peer ID")

	Debug("Test passed: peersInfoData is correct")
}

func TestDiscv5PeerMeshCount(t *testing.T) {
	Debug("Starting test to verify peer count in mesh using Discv5 after topic subscription")

	node1Config := DefaultWakuConfig
	node1Config.Relay = true
	Debug("Creating Node1")
	node1, err := StartWakuNode("Node1", &node1Config)
	require.NoError(t, err, "Failed to start Node1")

	enrNode1, err := node1.ENR()
	require.NoError(t, err, "Failed to get ENR for Node1")

	node2Config := DefaultWakuConfig
	node2Config.Discv5BootstrapNodes = []string{enrNode1.String()}
	node2Config.Relay = true
	Debug("Creating Node2 with Node1 as Discv5 bootstrap")
	node2, err := StartWakuNode("Node2", &node2Config)
	require.NoError(t, err, "Failed to start Node2")

	node3Config := DefaultWakuConfig
	node3Config.Discv5BootstrapNodes = []string{enrNode1.String()}
	node3Config.Relay = true

	Debug("Creating Node3 with Node2 as Discv5 bootstrap")
	node3, err := StartWakuNode("Node3", &node3Config)
	require.NoError(t, err, "Failed to start Node3")

	defer func() {
		Debug("Stopping and destroying all Waku nodes")
		node1.StopAndDestroy()
		node2.StopAndDestroy()
	}()

	defaultPubsubTopic := DefaultPubsubTopic
	Debug("Default pubsub topic retrieved: %s", defaultPubsubTopic)

	err = SubscribeNodesToTopic([]*WakuNode{node1, node2, node3}, defaultPubsubTopic)
	require.NoError(t, err, "Failed to subscribe all nodes to the topic")

	Debug("Waiting for nodes to auto-connect via Discv5")
	err = WaitForAutoConnection([]*WakuNode{node1, node2, node3})
	require.NoError(t, err, "Nodes did not auto-connect within timeout")
	time.Sleep(time.Second * 5)

	Debug("Fetching number of peers in mesh for Node1 before stopping Node3")
	peerCountBefore, err := node1.GetNumPeersInMesh(defaultPubsubTopic)
	require.NoError(t, err, "Failed to get number of peers in mesh for Node1 before stopping Node3")

	Debug("Total number of peers in mesh for Node1 before stopping Node3: %d", peerCountBefore)
	require.Equal(t, 2, peerCountBefore, "Expected Node1 to have exactly 2 peers in the mesh before stopping Node3")

	Debug("Stopping Node3")
	node3.StopAndDestroy()

	Debug("Waiting for network update after Node3 stops")
	time.Sleep(10 * time.Second)

	Debug("Fetching number of peers in mesh for Node1 after stopping Node3")
	peerCountAfter, err := node1.GetNumPeersInMesh(defaultPubsubTopic)
	require.NoError(t, err, "Failed to get number of peers in mesh for Node1 after stopping Node3")

	Debug("Total number of peers in mesh for Node1 after stopping Node3: %d", peerCountAfter)
	require.Equal(t, 1, peerCountAfter, "Expected Node1 to have exactly 1 peer in the mesh after stopping Node3")

	Debug("Test successfully verified peer count change after stopping Node3")
}

func TestDiscv5PeerMeshIds(t *testing.T) {
	Debug("Starting test to verify peers in mesh using Discv5 after topic subscription")

	node1Config := DefaultWakuConfig
	node1Config.Relay = true
	Debug("Creating Node1")
	node1, err := StartWakuNode("Node1", &node1Config)
	require.NoError(t, err, "Failed to start Node1")

	enrNode1, err := node1.ENR()
	require.NoError(t, err, "Failed to get ENR for Node1")

	node2Config := DefaultWakuConfig
	node2Config.Discv5BootstrapNodes = []string{enrNode1.String()}
	node2Config.Relay = true
	Debug("Creating Node2 with Node1 as Discv5 bootstrap")
	node2, err := StartWakuNode("Node2", &node2Config)
	require.NoError(t, err, "Failed to start Node2")

	node2PeerID, err := node2.PeerID()
	require.NoError(t, err, "Failed to get PeerID for Node 2")

	node3Config := DefaultWakuConfig
	node3Config.Discv5BootstrapNodes = []string{enrNode1.String()}
	node3Config.Relay = true

	Debug("Creating Node3 with Node2 as Discv5 bootstrap")
	node3, err := StartWakuNode("Node3", &node3Config)
	require.NoError(t, err, "Failed to start Node3")

	node3PeerID, err := node3.PeerID()
	require.NoError(t, err, "Failed to get PeerID for Node 3")

	defer func() {
		Debug("Stopping and destroying all Waku nodes")
		node1.StopAndDestroy()
		node2.StopAndDestroy()
	}()

	defaultPubsubTopic := DefaultPubsubTopic
	Debug("Default pubsub topic retrieved: %s", defaultPubsubTopic)

	err = SubscribeNodesToTopic([]*WakuNode{node1, node2, node3}, defaultPubsubTopic)
	require.NoError(t, err, "Failed to subscribe all nodes to the topic")

	Debug("Waiting for nodes to auto-connect via Discv5")
	err = WaitForAutoConnection([]*WakuNode{node1, node2, node3})
	require.NoError(t, err, "Nodes did not auto-connect within timeout")

	Debug("Fetching number of peers in mesh for Node1 before stopping Node3")
	peersBefore, err := node1.GetPeersInMesh(defaultPubsubTopic)
	require.NoError(t, err, "Failed to get number of peers in mesh for Node1 before stopping Node3")

	Debug("Total number of peers in mesh for Node1 before stopping Node3: %d", len(peersBefore))
	require.Equal(t, 2, len(peersBefore), "Expected Node1 to have exactly 2 peers in the mesh before stopping Node3")
	require.True(t, slices.Contains(peersBefore, node2PeerID), "Node 2 should be included in node 1's mesh")
	require.True(t, slices.Contains(peersBefore, node3PeerID), "Node 3 should be included in node 1's mesh")

	Debug("Stopping Node3")
	node3.StopAndDestroy()

	Debug("Waiting for network update after Node3 stops")
	time.Sleep(10 * time.Second)

	Debug("Fetching number of peers in mesh for Node1 after stopping Node3")
	peersAfter, err := node1.GetPeersInMesh(defaultPubsubTopic)
	require.NoError(t, err, "Failed to get number of peers in mesh for Node1 after stopping Node3")

	Debug("Total number of peers in mesh for Node1 after stopping Node3: %d", len(peersAfter))
	require.Equal(t, 1, len(peersAfter), "Expected Node1 to have exactly 1 peer in the mesh after stopping Node3")
	require.True(t, slices.Contains(peersBefore, node2PeerID), "Node 2 should be included in node 1's mesh")

	Debug("Test successfully verified peer count change after stopping Node3")
}

func TestDiscv5DisabledNoPeersConnected(t *testing.T) {
	Debug("Starting TestDiscv5DisabledNoPeersConnected")

	nodeConfig := DefaultWakuConfig
	nodeConfig.Discv5Discovery = false
	nodeConfig.Relay = true

	Debug("Creating Node1")
	node1, err := StartWakuNode("Node1", &nodeConfig)
	require.NoError(t, err, "Failed to start Node1")

	enrNode1, err := node1.ENR()
	require.NoError(t, err, "Failed to get ENR for Node1")
	nodeConfig.Discv5BootstrapNodes = []string{enrNode1.String()}

	Debug("Creating Node2 with Node1 as Discv5 bootstrap")
	node2, err := StartWakuNode("Node2", &nodeConfig)
	require.NoError(t, err, "Failed to start Node2")

	defer func() {
		Debug("Stopping and destroying all Waku nodes")
		node1.StopAndDestroy()
		node2.StopAndDestroy()
	}()

	Debug("Waiting to ensure no auto-connection")
	time.Sleep(15 * time.Second)

	Debug("Verifying number of peers connected to Nodes")
	peerCount, err := node1.GetNumConnectedPeers()
	require.NoError(t, err, "Failed to get number of peers in mesh for Node1")
	Debug("Total number of connected peers for Node1: %d", peerCount)
	require.Equal(t, 0, peerCount, "Expected Node1 to have exactly 0 peers in the mesh")

	peerCount, err = node2.GetNumConnectedPeers()
	require.NoError(t, err, "Failed to get number of peers in mesh for Node2")
	Debug("Total number of connected peers for Node2: %d", peerCount)
	require.Equal(t, 0, peerCount, "Expected Node2 to have exactly 0 peers in the mesh")

	Debug("Test passed: all the nodes have 0 peers")
}

// this test commented as it will fail will be changed to have external ip in future task
/*
func TestDiscv5GetPeersConnected(t *testing.T) {
	Debug("Starting test to verify peer count in mesh with 4 nodes using Discv5 (Chained Connection)")

	node1Config := DefaultWakuConfig
	node1Config.Relay = true

	Debug("Creating Node1")
	node1, err := StartWakuNode("Node1", &node1Config)
	require.NoError(t, err, "Failed to start Node1")

	enrNode1, err := node1.ENR()
	require.NoError(t, err, "Failed to get ENR for Node1")

	node2Config := DefaultWakuConfig
	node2Config.Relay = true
	node2Config.Discv5BootstrapNodes = []string{enrNode1.String()}

	Debug("Creating Node2 with Node1 as Discv5 bootstrap")
	node2, err := StartWakuNode("Node2", &node2Config)
	require.NoError(t, err, "Failed to start Node2")

	enrNode2, err := node2.ENR()
	require.NoError(t, err, "Failed to get ENR for Node2")

	node3Config := DefaultWakuConfig
	node3Config.Relay = true
	node3Config.Discv5BootstrapNodes = []string{enrNode2.String()}

	Debug("Creating Node3 with Node2 as Discv5 bootstrap")
	node3, err := StartWakuNode("Node3", &node3Config)
	require.NoError(t, err, "Failed to start Node3")

	enrNode3, err := node3.ENR()
	require.NoError(t, err, "Failed to get ENR for Node3")

	node4Config := DefaultWakuConfig
	node4Config.Relay = true
	node4Config.Discv5BootstrapNodes = []string{enrNode3.String()}

	Debug("Creating Node4 with Node3 as Discv5 bootstrap")
	node4, err := StartWakuNode("Node4", &node4Config)
	require.NoError(t, err, "Failed to start Node4")

	defer func() {
		Debug("Stopping and destroying all Waku nodes")
		node1.StopAndDestroy()
		node2.StopAndDestroy()
		node3.StopAndDestroy()
		node4.StopAndDestroy()
	}()

	Debug("Waiting for nodes to auto-connect via Discv5")
	err = WaitForAutoConnection([]*WakuNode{node1, node2, node3, node4})
	require.NoError(t, err, "Nodes did not auto-connect within timeout")

	Debug("Fetching number of peers in connected to  Node1")
	peerCount, err := node1.GetNumConnectedPeers()
	require.NoError(t, err, "Failed to get number of peers in mesh for Node1")

	Debug("Total number of peers connected to Node1: %d", peerCount)
	require.Equal(t, 3, peerCount, "Expected Node1 to have exactly 3 peers in the mesh")

	Debug("Test successfully verified peer count in mesh with 4 nodes using Discv5 (Chained Connection)")
}
*/
