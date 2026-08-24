package kernel

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/logos-messaging/logos-delivery-go-bindings/internal/ffi"
	"github.com/logos-messaging/logos-delivery-go-bindings/pkg/kernel/common"
)

// Store is a Node's store protocol surface. Take one with Node.Store.
type Store struct{ n *Node }

// Query runs a store query against a store peer and returns its response. A
// ctx without a deadline gets the package default of 30s.
func (s *Store) Query(
	ctx context.Context, request *common.StoreQueryRequest, peerInfo peer.AddrInfo,
) (*common.StoreQueryResponse, error) {
	if err := s.n.check(); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	requestJSON, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	addr, err := peerAddr(peerInfo)
	if err != nil {
		return nil, err
	}

	responseJSON, err := ffi.StoreQuery(
		s.n.h, string(requestJSON), addr, timeoutMillis(ctx, requestTimeout),
	)
	if err != nil {
		return nil, fmt.Errorf("kernel: store query: %w", err)
	}

	var response common.StoreQueryResponse
	if err := json.Unmarshal([]byte(responseJSON), &response); err != nil {
		return nil, err
	}
	return &response, nil
}
