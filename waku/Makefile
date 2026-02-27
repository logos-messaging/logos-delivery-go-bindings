# Makefile for Waku Go Bindings

# Path to logos-delivery submodule
export LMN_DIR ?= $(shell pwd)/../vendor/logos-delivery

# Debugging output
print-paths:
	@echo "LMN_DIR: $(LMN_DIR)"
	@echo "HEADER_PATH: $(LIBWAKU_HEADER_PATH)"
	@echo "LIB_PATH: $(LIBWAKU_LIB_PATH)"

# Default paths for libwaku library and headers (can be overridden)
export LIBWAKU_HEADER_PATH ?= $(LMN_DIR)/library
export LIBWAKU_LIB_PATH ?= $(LMN_DIR)/build

export CGO_CFLAGS := -I$(LIBWAKU_HEADER_PATH)/
export CGO_LDFLAGS := -L$(LIBWAKU_LIB_PATH)/ -lwaku -Wl,-rpath,$(LIBWAKU_LIB_PATH)/

# Expected files
HEADER_FILE := $(LIBWAKU_HEADER_PATH)/libwaku.h
LIB_FILES   := $(wildcard $(LIBWAKU_LIB_PATH)/libwaku.*)

.PHONY: all clean prepare build test build-auto test-auto build-libwaku

# Validate necessary folders and files
check-folders:
	@echo Checking libwaku header directory ...
	@if [ -z "$(LIBWAKU_HEADER_PATH)" ]; then \
		echo "ERROR: LIBWAKU_HEADER_PATH not set"; exit 1; \
	fi
	@if [ ! -d "$(LIBWAKU_HEADER_PATH)" ]; then \
		echo "ERROR: Header path does not exist: $(LIBWAKU_HEADER_PATH)"; exit 1; \
	fi

	@echo Checking libwaku lib directory ...
	@if [ -z "$(LIBWAKU_LIB_PATH)" ]; then \
		echo "ERROR: LIBWAKU_LIB_PATH not set"; exit 1; \
	fi
	@if [ ! -d "$(LIBWAKU_LIB_PATH)" ]; then \
		echo "ERROR: Library path does not exist: $(LIBWAKU_LIB_PATH)"; exit 1; \
	fi

	@echo Checking for libwaku.h ...
	@if [ ! -f "$(HEADER_FILE)" ]; then \
		echo "ERROR: libwaku.h not found at: $(HEADER_FILE)"; exit 1; \
	fi

	@echo Checking for libwaku library file ...
	@if [ -z "$(LIB_FILES)" ]; then \
		echo "ERROR: No libwaku library file found in: $(LIBWAKU_LIB_PATH)"; exit 1; \
	fi

build:
	@echo "Building Logos Messaging Go Bindings (manual)..."
	CGO_CFLAGS="$(CGO_CFLAGS)" CGO_LDFLAGS="$(CGO_LDFLAGS)" go build -o waku-bindings .

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
	@rm -f waku-bindings
