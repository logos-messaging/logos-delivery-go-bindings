package kernel

import (
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

// Several kernel endpoints build their reply by stringifying libp2p objects in
// modules that do not import libp2p's `$` for them, so Nim falls back to its
// default object rendering and the reply carries the object's raw bytes rather
// than the encoded value: a PeerId arrives as "(data: @[0, 37, ...])" and a
// MultiAddress as "(data: <hex>)". The decoders below accept that shape as well
// as the intended one, so they keep working once the library is fixed.

var (
	peerIDBytesPattern    = regexp.MustCompile(`\(data: @\[([0-9, ]*)\]\)`)
	multiaddrBytesPattern = regexp.MustCompile(`\(data: ([0-9A-Fa-f]*)\)`)
)

// decodePeerIDs parses the comma-separated peer ID list a kernel call returns.
func decodePeerIDs(peersStr string) (peer.IDSlice, error) {
	peersStr = strings.TrimSpace(peersStr)
	if peersStr == "" {
		return nil, nil
	}

	if matches := peerIDBytesPattern.FindAllStringSubmatch(peersStr, -1); len(matches) > 0 {
		peers := make(peer.IDSlice, 0, len(matches))
		for _, match := range matches {
			id, err := peerIDFromByteList(match[1])
			if err != nil {
				return nil, err
			}
			peers = append(peers, id)
		}
		return peers, nil
	}

	var peers peer.IDSlice
	for _, encoded := range strings.Split(peersStr, ",") {
		encoded = strings.TrimSpace(encoded)
		if encoded == "" {
			continue
		}
		id, err := peer.Decode(encoded)
		if err != nil {
			return nil, fmt.Errorf("failed to decode peer ID %q: %w", encoded, err)
		}
		peers = append(peers, id)
	}

	return peers, nil
}

func peerIDFromByteList(list string) (peer.ID, error) {
	raw := make([]byte, 0, len(list)/3)
	for _, field := range strings.Split(list, ",") {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		value, err := strconv.ParseUint(field, 10, 8)
		if err != nil {
			return "", fmt.Errorf("failed to parse peer ID byte %q: %w", field, err)
		}
		raw = append(raw, byte(value))
	}

	id, err := peer.IDFromBytes(raw)
	if err != nil {
		return "", fmt.Errorf("failed to build peer ID from %d bytes: %w", len(raw), err)
	}

	return id, nil
}

// decodeMultiaddrs parses the comma-separated multiaddress list a kernel call
// returns.
func decodeMultiaddrs(addrsStr string) ([]multiaddr.Multiaddr, error) {
	var addrs []multiaddr.Multiaddr
	for _, addr := range strings.Split(addrsStr, ",") {
		if strings.TrimSpace(addr) == "" {
			continue
		}

		decoded, err := decodeMultiaddr(addr)
		if err != nil {
			return nil, err
		}
		addrs = append(addrs, decoded)
	}

	return addrs, nil
}

// decodeMultiaddr parses one multiaddress from a kernel reply.
func decodeMultiaddr(addr string) (multiaddr.Multiaddr, error) {
	addr = strings.TrimSpace(addr)

	match := multiaddrBytesPattern.FindStringSubmatchIndex(addr)
	if match == nil {
		return multiaddr.NewMultiaddr(addr)
	}

	raw, err := hex.DecodeString(addr[match[2]:match[3]])
	if err != nil {
		return nil, fmt.Errorf("failed to decode multiaddr bytes: %w", err)
	}

	decoded, err := multiaddr.NewMultiaddrBytes(raw)
	if err != nil {
		return nil, fmt.Errorf("failed to build multiaddr from %d bytes: %w", len(raw), err)
	}

	if suffix := addr[match[1]:]; suffix != "" {
		suffixAddr, err := multiaddr.NewMultiaddr(suffix)
		if err != nil {
			return nil, fmt.Errorf("failed to parse multiaddr suffix %q: %w", suffix, err)
		}
		decoded = decoded.Encapsulate(suffixAddr)
	}

	return decoded, nil
}
