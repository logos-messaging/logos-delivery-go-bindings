# Makefile for Waku Go Bindings

# Path to logos-delivery submodule
export LOGOS_DELIVERY_DIR ?= $(shell pwd)/../../vendor/logos-delivery

# Debugging output
print-paths:
	@echo "LOGOS_DELIVERY_DIR: $(LOGOS_DELIVERY_DIR)"
	@echo "HEADER_PATH: $(LIBWAKU_HEADER_PATH)"
	@echo "LIB_PATH: $(LIBWAKU_LIB_PATH)"

# Default paths for the liblogosdelivery library and headers (can be
# overridden). Since logos-delivery#3949 the waku_* ABI ships inside the
# unified liblogosdelivery, so library/ is gone and -lwaku no longer exists.
export LIBWAKU_HEADER_PATH ?= $(LOGOS_DELIVERY_DIR)/liblogosdelivery
export LIBWAKU_LIB_PATH ?= $(LOGOS_DELIVERY_DIR)/build

export CGO_CFLAGS := -I$(LIBWAKU_HEADER_PATH)/
export CGO_LDFLAGS := -L$(LIBWAKU_LIB_PATH)/ -llogosdelivery -Wl,-rpath,$(LIBWAKU_LIB_PATH)/

# Expected files
HEADER_FILE := $(LIBWAKU_HEADER_PATH)/liblogosdelivery.h
LIB_FILES   := $(wildcard $(LIBWAKU_LIB_PATH)/liblogosdelivery.*)

.PHONY: all clean prepare build test build-auto test-auto build-libwaku

# Validate necessary folders and files
check-folders:
	@echo Checking liblogosdelivery header directory ...
	@if [ -z "$(LIBWAKU_HEADER_PATH)" ]; then \
		echo "ERROR: LIBWAKU_HEADER_PATH not set"; exit 1; \
	fi
	@if [ ! -d "$(LIBWAKU_HEADER_PATH)" ]; then \
		echo "ERROR: Header path does not exist: $(LIBWAKU_HEADER_PATH)"; exit 1; \
	fi

	@echo Checking liblogosdelivery lib directory ...
	@if [ -z "$(LIBWAKU_LIB_PATH)" ]; then \
		echo "ERROR: LIBWAKU_LIB_PATH not set"; exit 1; \
	fi
	@if [ ! -d "$(LIBWAKU_LIB_PATH)" ]; then \
		echo "ERROR: Library path does not exist: $(LIBWAKU_LIB_PATH)"; exit 1; \
	fi

	@echo Checking for liblogosdelivery.h ...
	@if [ ! -f "$(HEADER_FILE)" ]; then \
		echo "ERROR: liblogosdelivery.h not found at: $(HEADER_FILE)"; exit 1; \
	fi

	@echo Checking for liblogosdelivery library file ...
	@if [ -z "$(LIB_FILES)" ]; then \
		echo "ERROR: No liblogosdelivery library file found in: $(LIBWAKU_LIB_PATH)"; exit 1; \
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
