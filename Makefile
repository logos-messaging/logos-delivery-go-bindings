# Nimble resolves logos-delivery and owns the library build; Go resolves the Go
# side; this Makefile links them together.

NIMBLE ?= nimble
GO ?= go

LOGOS_DELIVERY_DIR = $(shell $(NIMBLE) path logos_delivery 2>/dev/null | tail -1)
LIB_DIR ?= $(CURDIR)/build
LIB_EXT ?= $(if $(filter Windows_NT,$(OS)),dll,$(if $(filter Darwin,$(shell uname -s)),dylib,so))
LIB := $(LIB_DIR)/liblogosdelivery.$(LIB_EXT)

# Windows has no -rpath; the loader uses PATH.
ifneq ($(LIB_EXT),dll)
    LIB_RPATH := -Wl,-rpath,$(LIB_DIR)
endif

export CGO_CFLAGS  = -I$(LOGOS_DELIVERY_DIR)/library
export CGO_LDFLAGS = -L$(LIB_DIR) -llogosdelivery $(LIB_RPATH)

.PHONY: deps liblogosdelivery liblogosdelivery-android liblogosdelivery-ios build example test clean print-paths print-cgo

deps: nimble.paths ##@build Resolve the Nim dependencies

nimble.paths: logos_delivery_go_bindings.nimble
	# nimble path lists every installed version, so keep only this resolution's.
	rm -rf nimbledeps
	$(NIMBLE) setup --localdeps -y

# Plain nim, so logos-delivery builds from this resolution and not from its own
# lock. nimble shell puts the Nim this resolution selected on PATH.
NIM_TASK = NIM_PARAMS="--skipParentCfg $$NIM_PARAMS -d:disable_rln $$(tr '\n' ' ' < nimble.paths)" \
	$(NIMBLE) shell nim --skipParentCfg
NIM_ENTRY = "$$($(NIMBLE) path logos_delivery)/liblogosdelivery.nims"

$(LIB): | nimble.paths
	LIB_OUT_DIR="$(LIB_DIR)" $(NIM_TASK) liblogosdelivery $(NIM_ENTRY)
	@test -f $@ || (echo "ERROR: $@ was not produced" && exit 1)

liblogosdelivery: $(LIB) ##@build Build liblogosdelivery from the Nimble dependency

liblogosdelivery-android: | nimble.paths ##@build Build liblogosdelivery for Android; set ANDROID_NDK_ROOT, CPU and ABIDIR
	LIB_OUT_DIR="$(LIB_DIR)/android/$(ABIDIR)" $(NIM_TASK) libLogosDeliveryAndroid $(NIM_ENTRY)

liblogosdelivery-ios: | nimble.paths ##@build Build liblogosdelivery for iOS; set IOS_SDK, IOS_ARCH and IOS_SDK_PATH
	LIB_OUT_DIR="$(LIB_DIR)/ios/$(IOS_SDK)-$(IOS_ARCH)" $(NIM_TASK) libLogosDeliveryIOS $(NIM_ENTRY)

build: $(LIB) ##@build Build the Go packages
	$(GO) build ./...

example: $(LIB) ##@run Run the messaging example
	$(GO) run ./examples/messaging

test: $(LIB) ##@test Run the Go tests; TEST=<name> to select one
	@if [ -z "$(TEST)" ]; then \
		$(GO) test ./...; \
	else \
		$(GO) test ./... -count=1 -run $(TEST) -v; \
	fi

print-paths:
	@echo "LOGOS_DELIVERY_DIR: $(LOGOS_DELIVERY_DIR)"
	@echo "LIB:                $(LIB)"

print-cgo: ##@build Print the cgo flags, for a caller that runs Go itself
	@echo 'CGO_CFLAGS=$(CGO_CFLAGS)'
	@echo 'CGO_LDFLAGS=$(CGO_LDFLAGS)'

clean:
	@rm -f $(LIB)
