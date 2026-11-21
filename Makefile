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

# The Nim this resolution installed, else the one on PATH.
NIM ?= $(or $(firstword $(wildcard $(CURDIR)/nimbledeps/pkgs2/nim-*/bin/nim $(CURDIR)/nimbledeps/pkgs2/nim-*/bin/nim.exe)),nim)

# Leopard-RS's CMake flags: portable, position-independent, no OpenMP. On x86 it
# includes SSSE3 intrinsics unconditionally.
LEOPARD_FLAGS = -DCMAKE_BUILD_TYPE=Release -DENABLE_OPENMP=off -DCOMPILER_SUPPORTS_MARCH_NATIVE=FALSE -DCMAKE_POSITION_INDEPENDENT_CODE=ON
LEOPARD_X86 = $(if $(filter x86_64 amd64,$(1)),-DCMAKE_CXX_FLAGS=-mssse3)
LEOPARD_DESKTOP = $(if $(filter dll,$(LIB_EXT)),,$(LEOPARD_FLAGS) $(call LEOPARD_X86,$(shell uname -m)))

# Plain nim c on logos-delivery's library entry, from this resolution and not
# from logos-delivery's own lock. nimble.paths has every --path, so parent
# configs are skipped; library/config.nims, next to the entry, is still read.
# eval: nimble.paths and the flags quote their arguments.
# $(call NIM_BUILD,<output dir>,<lib|staticlib>,<file>,<Leopard-RS CMake flags>,<flags>)
NIM_BUILD = mkdir -p "$(1)" && eval "\"$(NIM)\" c --skipParentCfg $$(tr '\r\n' '  ' < nimble.paths) \
	-d:disable_rln --app:$(2) $(5) $(if $(strip $(4)),-d:LeopardCmakeFlags=\"$(strip $(4))\") $(subst ",\",$(NIM_PARAMS)) \
	--nimcache:\"$(1)/nimcache\" -d:ffiOutputDir=\"$(1)/generated\" -o:\"$(1)/$(3)\" \
	\"$(LOGOS_DELIVERY_DIR)/library/liblogosdelivery.nim\""

$(LIB): | nimble.paths
	$(call NIM_BUILD,$(LIB_DIR),lib,$(notdir $(LIB)),$(LEOPARD_DESKTOP))
	@test -f $@ || (echo "ERROR: $@ was not produced" && exit 1)

liblogosdelivery: $(LIB) ##@build Build liblogosdelivery from the Nimble dependency

ANDROID_BIN = $(ANDROID_NDK_ROOT)/toolchains/llvm/prebuilt/$(if $(filter Darwin,$(shell uname -s)),darwin,linux)-x86_64/bin
ANDROID_TRIPLE = $(if $(filter arm64,$(CPU)),aarch64-linux-android,$(if $(filter arm,$(CPU)),armv7a-linux-androideabi,$(if $(filter i386,$(CPU)),i686-linux-android,x86_64-linux-android)))
ANDROID_CC = $(ANDROID_BIN)/$(ANDROID_TRIPLE)$(or $(ANDROID_API),28)-clang

liblogosdelivery-android: | nimble.paths ##@build Build liblogosdelivery for Android; set ANDROID_NDK_ROOT, CPU and ABIDIR
	$(call NIM_BUILD,$(LIB_DIR)/android/$(ABIDIR),lib,liblogosdelivery.so,\
		$(LEOPARD_FLAGS) -DCMAKE_SYSTEM_NAME=Linux -DCMAKE_C_COMPILER=$(ANDROID_CC) -DCMAKE_CXX_COMPILER=$(ANDROID_CC)++ \
		-DCMAKE_CXX_FLAGS=-DANDROID -DCMAKE_AR=$(ANDROID_BIN)/llvm-ar -DCMAKE_RANLIB=$(ANDROID_BIN)/llvm-ranlib,\
		--os:android --cpu:$(CPU) --cc:clang --clang.exe:$(ANDROID_CC) --clang.linkerexe:$(ANDROID_CC) \
		-d:LeopardExtraCompilerFlags=-fno-openmp -d:LeopardExtraLinkerFlags=-fno-openmp -d:chronicles_log_level=ERROR)

IOS_DEPLOYMENT_TARGET ?= 13.0
IOS_TARGET_FLAGS = -arch $(IOS_ARCH) -isysroot $(IOS_SDK_PATH) \
	$(if $(filter iphonesimulator,$(IOS_SDK)),-mios-simulator-version-min,-mios-version-min)=$(IOS_DEPLOYMENT_TARGET)

liblogosdelivery-ios: | nimble.paths ##@build Build liblogosdelivery for iOS; set IOS_SDK, IOS_ARCH and IOS_SDK_PATH
	$(call NIM_BUILD,$(LIB_DIR)/ios/$(IOS_SDK)-$(IOS_ARCH),staticlib,liblogosdelivery.a,\
		$(LEOPARD_FLAGS) -DCMAKE_SYSTEM_NAME=iOS -DCMAKE_OSX_SYSROOT=$(IOS_SDK_PATH) -DCMAKE_OSX_ARCHITECTURES=$(IOS_ARCH) \
		-DCMAKE_OSX_DEPLOYMENT_TARGET=$(IOS_DEPLOYMENT_TARGET) $(call LEOPARD_X86,$(IOS_ARCH)),\
		--os:ios --cpu:$(if $(filter arm64,$(IOS_ARCH)),arm64,amd64) --cc:clang \
		--passC:\"$(IOS_TARGET_FLAGS)\" --passL:\"$(IOS_TARGET_FLAGS)\" --opt:size -d:chronicles_log_level=ERROR)

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
