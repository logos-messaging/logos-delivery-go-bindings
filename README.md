# Logos Delivery Go Bindings

Go bindings for Logos Delivery.

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
go get github.com/logos-messaging/logos-delivery-go-bindings
```

## Building

`liblogosdelivery` is required at compile time. This repository is a Nimble package declaring the
minimum logos-delivery its C ABI works against, so a consumer picks the revision and can upgrade
without a release here.

```sh
nimble setup                # resolve logos-delivery at the pinned revision
nimble liblogosdelivery     # build it, via logos-delivery's own build task
```

The library is written to logos-delivery's package directory. Set `LIBLOGOSDELIVERY_OUT` to have it
copied somewhere of your choosing, and pass `NIM_PARAMS` for build defines — `-d:disable_rln` builds
without zerokit, so no Rust toolchain is needed:

```sh
LIBLOGOSDELIVERY_OUT="$PWD/build" NIM_PARAMS="-d:disable_rln" nimble liblogosdelivery
```

Then point cgo at the headers and the library:

```sh
export CGO_CFLAGS="-I$(nimble path logos_delivery | tail -1)/library"
export CGO_LDFLAGS="-L$PWD/build -llogosdelivery -Wl,-rpath,$PWD/build"

go build ./...
go test ./pkg/messaging/
```

## Development

No clone of logos-delivery is needed: `nimble setup` fetches it at the pinned revision.

To work against a local checkout instead — testing a logos-delivery change before repinning here —
shadow the resolved package:

```sh
nimble develop /path/to/logos-delivery
```

To choose a specific revision, require it from your own `.nimble` — it satisfies the range here and
wins resolution:

```nim
requires "https://github.com/logos-messaging/logos-delivery#<rev>"
```
