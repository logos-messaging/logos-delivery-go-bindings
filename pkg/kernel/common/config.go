package common

type WakuConfig struct {
	// Host is the LibP2P listening address; the consolidated WakuNodeConf calls
	// it listenAddress. This expects an IPv4.
	Host                        string           `json:"listenAddress,omitempty"`
	Nodekey                     string           `json:"nodekey,omitempty"`
	Relay                       bool             `json:"relay"`
	Store                       bool             `json:"store"`
	Storenode                   string           `json:"storenode,omitempty"`
	StoreMessageRetentionPolicy string           `json:"storeMessageRetentionPolicy,omitempty"`
	StoreMessageDbUrl           string           `json:"storeMessageDbUrl,omitempty"`
	StoreMessageDbVacuum        bool             `json:"storeMessageDbVacuum"`
	StoreMaxNumDbConnections    int              `json:"storeMaxNumDbConnections,omitempty"`
	StoreResume                 bool             `json:"storeResume"`
	Filter                      bool             `json:"filter"`
	Filternode                  string           `json:"filternode,omitempty"`
	FilterSubscriptionTimeout   int64            `json:"filterSubscriptionTimeout,omitempty"`
	FilterMaxPeersToServe       uint32           `json:"filterMaxPeersToServe,omitempty"`
	FilterMaxCriteria           uint32           `json:"filterMaxCriteria,omitempty"`
	Lightpush                   bool             `json:"lightpush"`
	LightpushNode               string           `json:"lightpushnode,omitempty"`
	LogLevel                    string           `json:"logLevel,omitempty"`
	DnsDiscovery                bool             `json:"dnsDiscovery"`
	DnsDiscoveryUrl             string           `json:"dnsDiscoveryUrl,omitempty"`
	MaxMessageSize              string           `json:"maxMessageSize,omitempty"`
	Staticnodes                 []string         `json:"staticnodes,omitempty"`
	Discv5BootstrapNodes        []string         `json:"discv5BootstrapNodes,omitempty"`
	Discv5Discovery             bool             `json:"discv5Discovery"`
	Discv5UdpPort               int              `json:"discv5UdpPort"`
	ClusterID                   uint16           `json:"clusterId,omitempty"`
	Shards                      []uint16         `json:"shards,omitempty"`
	PeerExchange                bool             `json:"peerExchange"`
	PeerExchangeNode            string           `json:"peerExchangeNode,omitempty"`
	TcpPort                     int              `json:"tcpPort"`
	RateLimits                  RateLimitsConfig `json:"rateLimits,omitempty"`
	DnsAddrsNameServers         []string         `json:"dnsAddrsNameServers,omitempty"`
	Discv5EnrAutoUpdate         bool             `json:"discv5EnrAutoUpdate"`
	MaxConnections              int              `json:"maxConnections,omitempty"`
	NumShardsInNetwork          uint16           `json:"numShardsInNetwork"`
}
