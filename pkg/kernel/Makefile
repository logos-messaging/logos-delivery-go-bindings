# Makefile for the kernel bindings. The root Makefile is the usual entry point;
# this one exists for building and testing just this package.

NIMBLE ?= nimble

# Resolved by Nimble, not vendored.
export LOGOS_DELIVERY_DIR ?= $(shell $(NIMBLE) path logos_delivery 2>/dev/null | tail -1)

# Debugging output
print-paths:
	@echo "LOGOS_DELIVERY_DIR: $(LOGOS_DELIVERY_DIR)"
	@echo "HEADER_PATH: $(LOGOS_DELIVERY_HEADER_PATH)"
	@echo "LIB_PATH: $(LOGOS_DELIVERY_LIB_PATH)"

# Default paths for the library and headers (can be overridden)
export LOGOS_DELIVERY_HEADER_PATH ?= $(LOGOS_DELIVERY_DIR)/library
export LOGOS_DELIVERY_LIB_PATH ?= $(CURDIR)/../../build

export CGO_CFLAGS := -I$(LOGOS_DELIVERY_HEADER_PATH)/
export CGO_LDFLAGS := -L$(LOGOS_DELIVERY_LIB_PATH)/ -llogosdelivery -Wl,-rpath,$(LOGOS_DELIVERY_LIB_PATH)/

# Expected files
HEADER_FILE := $(LOGOS_DELIVERY_HEADER_PATH)/liblogosdelivery.h
LIB_FILES   := $(wildcard $(LOGOS_DELIVERY_LIB_PATH)/liblogosdelivery.*)

.PHONY: all clean prepare build test build-auto test-auto

# Validate necessary folders and files
check-folders:
	@echo Checking header directory ...
	@if [ -z "$(LOGOS_DELIVERY_HEADER_PATH)" ]; then \
		echo "ERROR: LOGOS_DELIVERY_HEADER_PATH not set"; exit 1; \
	fi
	@if [ ! -d "$(LOGOS_DELIVERY_HEADER_PATH)" ]; then \
		echo "ERROR: Header path does not exist: $(LOGOS_DELIVERY_HEADER_PATH)"; exit 1; \
	fi

	@echo Checking library directory ...
	@if [ -z "$(LOGOS_DELIVERY_LIB_PATH)" ]; then \
		echo "ERROR: LOGOS_DELIVERY_LIB_PATH not set"; exit 1; \
	fi
	@if [ ! -d "$(LOGOS_DELIVERY_LIB_PATH)" ]; then \
		echo "ERROR: Library path does not exist: $(LOGOS_DELIVERY_LIB_PATH)"; exit 1; \
	fi

	@echo Checking for liblogosdelivery.h ...
	@if [ ! -f "$(HEADER_FILE)" ]; then \
		echo "ERROR: liblogosdelivery.h not found at: $(HEADER_FILE)"; exit 1; \
	fi

	@echo Checking for the library file ...
	@if [ -z "$(LIB_FILES)" ]; then \
		echo "ERROR: No library file found in: $(LOGOS_DELIVERY_LIB_PATH)"; exit 1; \
	fi

build:
	@echo "Building Logos Messaging Go Bindings (manual)..."
	CGO_CFLAGS="$(CGO_CFLAGS)" CGO_LDFLAGS="$(CGO_LDFLAGS)" go build -o kernel-bindings .

test: build
	@echo "Running tests (manual)..."
	@if [ -z "$(TEST)" ]; then \
		CGO_CFLAGS="$(CGO_CFLAGS)" CGO_LDFLAGS="$(CGO_LDFLAGS)" go test .; \
	else \
		CGO_CFLAGS="$(CGO_CFLAGS)" CGO_LDFLAGS="$(CGO_LDFLAGS)" go test . -count=1 -run $(TEST) -v; \
	fi

# Clean up generated files
clean:
	@echo "Cleaning up..."
	@rm -f kernel-bindings
