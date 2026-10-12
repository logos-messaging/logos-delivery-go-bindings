# Nimble resolves logos-delivery and owns the library build; Go resolves the Go
# side; this Makefile links them together.

NIMBLE ?= nimble
GO ?= go

LOGOS_DELIVERY_DIR = $(shell $(NIMBLE) path logos_delivery 2>/dev/null | tail -1)
LIB_DIR ?= $(CURDIR)/build
LIB_EXT ?= $(if $(filter Darwin,$(shell uname -s)),dylib,so)
LIB := $(LIB_DIR)/liblogosdelivery.$(LIB_EXT)

export CGO_CFLAGS  = -I$(LOGOS_DELIVERY_DIR)/library
export CGO_LDFLAGS = -L$(LIB_DIR) -llogosdelivery -Wl,-rpath,$(LIB_DIR)

.PHONY: deps liblogosdelivery liblogosdelivery-android liblogosdelivery-ios build example test clean print-paths

deps: nimble.paths ##@build Resolve the Nim dependencies

nimble.paths:
	$(NIMBLE) setup --localdeps -y

$(LIB): | nimble.paths
	LIBLOGOSDELIVERY_OUT="$(LIB_DIR)" NIM_PARAMS="$$NIM_PARAMS -d:disable_rln" \
		$(NIMBLE) liblogosdelivery
	@test -f $@ || (echo "ERROR: $@ was not produced" && exit 1)

liblogosdelivery: $(LIB) ##@build Build liblogosdelivery from the Nimble dependency

liblogosdelivery-android: | nimble.paths ##@build Build liblogosdelivery for Android; set ANDROID_NDK_ROOT, CPU and ABIDIR
	LIBLOGOSDELIVERY_OUT="$(LIB_DIR)/android/$(ABIDIR)" NIM_PARAMS="$$NIM_PARAMS -d:disable_rln" \
		$(NIMBLE) liblogosdeliveryAndroid

liblogosdelivery-ios: | nimble.paths ##@build Build liblogosdelivery for iOS; set IOS_SDK, IOS_ARCH and IOS_SDK_PATH
	LIBLOGOSDELIVERY_OUT="$(LIB_DIR)/ios/$(IOS_SDK)-$(IOS_ARCH)" NIM_PARAMS="$$NIM_PARAMS -d:disable_rln" \
		$(NIMBLE) liblogosdeliveryIOS

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

clean:
	@rm -f $(LIB)
