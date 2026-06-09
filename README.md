# logos-delivery Go Bindings

Go bindings for the Waku library.

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
make -C waku build

# run all tests
make -C waku test

# run a specific test
make -C waku test TEST=TestConnectedPeersInfo
```

## Development

When working on this repository itself, `logos-delivery` is included as a git submodule for convenience.

- Initialize and update the submodule, then build `libwaku`
    ```sh
    git submodule update --init --recursive
    make -C waku build-libwaku
    ```
- Build the project. Submodule paths are used by default to find `libwaku`.
    ```shell
    make -C waku build
    ```
