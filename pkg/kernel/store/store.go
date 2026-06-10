package store

import (
	"github.com/libp2p/go-libp2p/core/protocol"
)

const StoreQueryID_v300 = protocol.ID("/vac/waku/store-query/3.0.0")

type FilterCriteria struct {
	ContentFilter interface{}
	TimeStart     *int64
	TimeEnd       *int64
}
