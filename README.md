# logos-delivery Go Bindings

Go bindings for the Waku library.

## Packages

- **`pkg/messaging`** — the high-level Messaging API, and the one to reach for.
  A `MessagingClient` mirrors the Nim `MessagingClient`: it owns a node and
  exposes `Start`/`Stop`/`Close`, `Subscribe`/`Unsubscribe`, `Send`, and a
  typed `Events()` stream. See `examples/messaging` for a runnable demo.
- **`pkg/kernel`** — the low-level kernel (`waku_*`) protocols: relay, store,
  filter, lightpush, discovery and peer management. Unsupported and subject to
  change without notice.

## Install

```
go get -u github.com/logos-messaging/logos-delivery-go-bindings
```

## Building

`liblogosdelivery` is required at compile time, and the Makefile fetches and
builds it — a clone of this repository is all you need. No separate checkout of
logos-delivery, no Nix, no environment variables:

```sh
make build                            # build liblogosdelivery, then the Go packages
make test
make test TEST=TestConnectedPeersInfo
```

The first run fetches logos-delivery into `_third_party/` and builds it there;
later runs reuse it. The leading underscore keeps the go tool out of the
checkout, which carries Go files of its own. RLN is
stubbed out by default, so no Rust toolchain is needed — pass
`DISABLE_RLN=false` to link real RLN.

`make build` and `make test` set the cgo flags themselves. For a tool that runs
`go` on its own — a CI step, a linter, an IDE — take them from the Makefile
rather than repeating them:

```sh
make cgo-env            # CGO_CFLAGS=... / CGO_LDFLAGS=... on stdout
```

### The logos-delivery pin

`logos-delivery.rev` names the revision whose C ABI this repository's cgo layer
is written against. It is a property of the bindings, not a choice each consumer
makes, so CI and consumers read the pairing from one place instead of resolving
`master` and hoping.

To move it, and let CI prove the new pair works:

```sh
make repin      # writes the current tip of logos-delivery master
make test
```

### Consuming from a Go module

A module zip carries no submodules and Go runs no build step, so a consumer has
to produce `liblogosdelivery` itself. It does not have to know how: the module
ships this Makefile and the pin, and the module cache is only read from — every
artifact lands in the directory the consumer names.

```make
LOGOS_DELIVERY_BINDINGS := $(shell go list -m -f '{{.Dir}}' \
    github.com/logos-messaging/logos-delivery-go-bindings)
LOGOS_DELIVERY_DIR := $(CURDIR)/_third_party/logos-delivery

LOGOS_DELIVERY_INC_DIR := $(LOGOS_DELIVERY_DIR)/library
LOGOS_DELIVERY_LIB_DIR := $(LOGOS_DELIVERY_DIR)/build
CGO_CFLAGS  += -I$(LOGOS_DELIVERY_INC_DIR)
CGO_LDFLAGS += -L$(LOGOS_DELIVERY_LIB_DIR)

liblogosdelivery:
	$(MAKE) -C $(LOGOS_DELIVERY_BINDINGS) \
		LOGOS_DELIVERY_DIR=$(LOGOS_DELIVERY_DIR) liblogosdelivery
```

The revision is whatever `logos-delivery.rev` says in the version of the
bindings that `go.mod` resolves to, so the consumer inherits the C ABI pairing
from the module it already pins, and a `go get -u` of the bindings carries the
matching logos-delivery with it.

## Development

To work against a local logos-delivery checkout — testing a change there before
repinning here — point the build at it. Its git state is left alone; only a
checkout this Makefile created is fetched into.

```sh
make build LOGOS_DELIVERY_DIR=/absolute/path/to/logos-delivery
```
