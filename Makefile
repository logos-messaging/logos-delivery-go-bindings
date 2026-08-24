# Building these bindings needs liblogosdelivery. This Makefile fetches and
# builds it, so a clone of this repository is all that is required.
#
# The revision lives in logos-delivery.rev. It is the C ABI this repository's
# cgo layer is written against — a property of the bindings, not a choice each
# consumer makes — so CI, Nix and any consumer can read the pin from one place.

GO ?= go

LOGOS_DELIVERY_URL ?= https://github.com/logos-messaging/logos-delivery.git
LOGOS_DELIVERY_REV := $(shell cat $(CURDIR)/logos-delivery.rev)
# The leading underscore is load-bearing: the go tool ignores directories
# starting with `_` or `.`, so `go build ./...` does not walk into the checkout.
# (`vendor/` would be worse still -- that name puts Go into vendor mode.)
LOGOS_DELIVERY_DIR ?= $(CURDIR)/_third_party/logos-delivery
# Named after the pin, so a checkout restored from a CI cache is neither
# re-fetched nor rebuilt while the pin is unchanged. It sits beside the
# checkout, not under $(CURDIR): a consumer runs this Makefile from its
# read-only module cache, and everything written has to land in its tree.
REV_STAMP := $(dir $(LOGOS_DELIVERY_DIR)).logos-delivery-$(LOGOS_DELIVERY_REV)

# Set false to build a checkout you manage yourself -- your own clone, with
# your own branch checked out. Its git state is then left alone.
LOGOS_DELIVERY_FETCH ?= true
ifeq ($(LOGOS_DELIVERY_FETCH),true)
    LOGOS_DELIVERY_FETCH_DEP := $(REV_STAMP)
endif

# Link the RLN stubs instead of zerokit, so no Rust toolchain is needed. A node
# whose config enables RLN then fails to start; set false to link real RLN.
DISABLE_RLN ?= true

ifeq ($(shell uname -s),Darwin)
	LIB_EXT := dylib
else
	LIB_EXT := so
endif

LIBLOGOSDELIVERY := $(LOGOS_DELIVERY_DIR)/build/liblogosdelivery.$(LIB_EXT)

# The cgo bridge self-links -llogosdelivery through a #cgo directive, so only
# the search paths are set here. library/ holds both checked-in headers and the
# generated one they include.
export CGO_CFLAGS  := -I$(LOGOS_DELIVERY_DIR)/library
export CGO_LDFLAGS := -L$(LOGOS_DELIVERY_DIR)/build -Wl,-rpath,$(LOGOS_DELIVERY_DIR)/build

.PHONY: all fetch liblogosdelivery build test repin cgo-env print-paths clean distclean

all: build

# A shallow fetch of the pinned commit only. Re-runs when the pin changes;
# logos-delivery's own Makefile installs nim/nimble and resolves its Nim
# dependencies, so nothing else has to be provisioned.
$(REV_STAMP):
	@mkdir -p $(LOGOS_DELIVERY_DIR)
	@echo "Fetching logos-delivery $(LOGOS_DELIVERY_REV)"
	@test -d $(LOGOS_DELIVERY_DIR)/.git || ( \
		git -C $(LOGOS_DELIVERY_DIR) init -q && \
		git -C $(LOGOS_DELIVERY_DIR) remote add origin $(LOGOS_DELIVERY_URL))
	git -C $(LOGOS_DELIVERY_DIR) fetch -q --depth 1 origin $(LOGOS_DELIVERY_REV)
	git -C $(LOGOS_DELIVERY_DIR) checkout -q --detach FETCH_HEAD
	@rm -f $(dir $(LOGOS_DELIVERY_DIR)).logos-delivery-*
	@touch $@

$(LIBLOGOSDELIVERY): $(LOGOS_DELIVERY_FETCH_DEP)
	$(MAKE) -C $(LOGOS_DELIVERY_DIR) DISABLE_RLN=$(DISABLE_RLN) liblogosdelivery

fetch: $(REV_STAMP) ## Fetch the pinned logos-delivery checkout

liblogosdelivery: $(LIBLOGOSDELIVERY) ## Fetch and build the pinned liblogosdelivery

build: $(LIBLOGOSDELIVERY)
	$(GO) build ./...

test: $(LIBLOGOSDELIVERY)
	@if [ -z "$(TEST)" ]; then \
		$(GO) test ./...; \
	else \
		$(GO) test ./... -count=1 -run $(TEST) -v; \
	fi

# Move the pin to the current tip of logos-delivery master. Commit the result:
# the next build, here and in CI, is the one that proves the pair still works.
repin:
	@git ls-remote $(LOGOS_DELIVERY_URL) HEAD | cut -f1 > logos-delivery.rev
	@echo "logos-delivery pinned to $$(cat logos-delivery.rev)"

# KEY=value lines for callers that run go themselves — a CI step
# (`make cgo-env >> "$$GITHUB_ENV"`), a linter, an IDE. Not shell-quoted: the
# value of CGO_LDFLAGS contains spaces, so this is not `eval`-able.
cgo-env:
	@echo 'CGO_CFLAGS=$(CGO_CFLAGS)'
	@echo 'CGO_LDFLAGS=$(CGO_LDFLAGS)'

print-paths:
	@echo "LOGOS_DELIVERY_REV: $(LOGOS_DELIVERY_REV)"
	@echo "LOGOS_DELIVERY_DIR: $(LOGOS_DELIVERY_DIR)"
	@echo "LIBLOGOSDELIVERY:   $(LIBLOGOSDELIVERY)"

clean:
	$(GO) clean ./...
	@rm -f pkg/kernel/kernel-bindings pkg/kernel/waku-bindings

# The fetched checkout is a build artifact like any other.
distclean: clean
	@rm -rf $(CURDIR)/_third_party
