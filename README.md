# logos-delivery Go Bindings

Go bindings for the Waku library.

## Install

```
go get -u github.com/logos-messaging/logos-delivery-go-bindings
```

## Building & Dependencies

`liblogosdelivery` (from `logos-delivery`) is required at compile-time. Since
logos-delivery#3949 it is a single library exposing both the `waku_*` Kernel
API and the `logosdelivery_*` Messaging API.

### Building with Makefile

If you have `logos-delivery` checked out, point the build to it:

```bash
# path to your existing logos-delivery clone
export LOGOS_DELIVERY_DIR=/absolute/path/to/logos-delivery
export CGO_CFLAGS="-I${LOGOS_DELIVERY_DIR}/liblogosdelivery"
export CGO_LDFLAGS="-L${LOGOS_DELIVERY_DIR}/build -llogosdelivery -Wl,-rpath,${LOGOS_DELIVERY_DIR}/build"

# compile all packages
make -C pkg/kernel build

# run all tests
make -C pkg/kernel test

# run a specific test
make -C pkg/kernel test TEST=TestConnectedPeersInfo
```

## Development

When working on this repository itself, `logos-delivery` is included as a git submodule for convenience.

- Initialize and update the submodule, then build `liblogosdelivery`
    ```sh
    git submodule update --init --recursive
    make -C vendor/logos-delivery liblogosdelivery
    ```
- Build the project. Submodule paths are used by default to find `liblogosdelivery`.
    ```shell
    make -C pkg/kernel build
    ```
