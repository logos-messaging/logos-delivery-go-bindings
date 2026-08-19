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

## Building & Dependencies

`libwaku` (from `logos-delivery`) is required at compile-time.

### Building with Makefile

If you have `logos-delivery` checked out, point the build to it:

```bash
# path to your existing logos-delivery clone
export LOGOS_DELIVERY_DIR=/absolute/path/to/logos-delivery
export CGO_CFLAGS="-I${LOGOS_DELIVERY_DIR}/library"
export CGO_LDFLAGS="-L${LOGOS_DELIVERY_DIR}/build -lwaku -Wl,-rpath,${LOGOS_DELIVERY_DIR}/build"

# compile all packages
make -C pkg/kernel build

# run all tests
make -C pkg/kernel test

# run a specific test
make -C pkg/kernel test TEST=TestConnectedPeersInfo
```

## Development

When working on this repository itself, `logos-delivery` is included as a git submodule for convenience.

- Initialize and update the submodule, then build `libwaku`
    ```sh
    git submodule update --init --recursive
    make -C pkg/kernel build-libwaku
    ```
- Build the project. Submodule paths are used by default to find `libwaku`.
    ```shell
    make -C pkg/kernel build
    ```
