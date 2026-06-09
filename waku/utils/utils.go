package utils

import (
	"runtime"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

func GetRSSKB() (uint64, error) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.Sys / 1024, nil
}

func EncapsulatePeerID(p peer.ID, addrs ...multiaddr.Multiaddr) []multiaddr.Multiaddr {
	encapsulated := make([]multiaddr.Multiaddr, 0, len(addrs))
	for _, addr := range addrs {
		encapsulated = append(encapsulated, addr.Encapsulate(multiaddr.StringCast("/p2p/"+p.String())))
	}
	return encapsulated
}
