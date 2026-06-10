package messaging

import "github.com/logos-messaging/logos-delivery-go-bindings/pkg/kernel/common"

// Config is the node configuration passed to the underlying library. Its JSON
// representation is a WakuNodeConf, which is what logosdelivery_create_node
// consumes. It aliases the kernel config type so the two stay in lockstep.
type Config = common.WakuConfig
