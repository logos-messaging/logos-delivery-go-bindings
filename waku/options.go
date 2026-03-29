package waku

import "github.com/waku-org/waku-go-bindings/waku/common"

type WakuNodeOption func(*WakuNode)

// This allows you to pass arbitrary valid config options to Nwaku based on https://github.com/waku-org/nwaku/blob/master/waku/factory/external_config.nim
// It is mostly for development and experimental purposes and will be removed in the future.
func WithExtraOptions(extraOptions common.ExtraOptions) WakuNodeOption {
	return func(wn *WakuNode) {
		wn.extraOptions = extraOptions
	}
}
