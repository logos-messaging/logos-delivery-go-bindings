mode = ScriptMode.Verbose

import std/[os, strutils]

### Package
version     = "0.1.0"
author      = "Logos"
description = "Go bindings for Logos Delivery"
license     = "MIT or Apache License 2.0"

# This is a Go module. The Nimble package exists so consumers get a
# liblogosdelivery whose C ABI matches these bindings.
#
# Pinned to a commit because logos-delivery's version does not move per release
# yet: every revision is 0.38.1, so a range cannot express "at least the one
# that has the liblogosdelivery task". Once it publishes versioned tags this
# becomes a range, and consumers can then upgrade without a release here.
#
# srcDir points at an empty directory and the Go trees are skipped, so nothing
# is contributed to a dependent's Nim path.
srcDir = "internal/nimble/src"

skipDirs = @["pkg", "internal", "examples", "tools", "nimble"]

### Dependencies
requires "nim >= 2.2.4"
requires "https://github.com/logos-messaging/logos-delivery#de59dcb2"

### Helpers

proc nimblePkgDir(name: string): string =
  ## Where the dependency was installed. `nimble path` prints one line per
  ## installed version and exits 0 even when the package is missing, so take the
  ## line that holds the package's nimble file rather than trusting position.
  let (output, _) = gorgeEx("nimble path " & name)
  for line in output.strip().splitLines():
    let candidate = line.strip()
    if candidate.isAbsolute() and fileExists(candidate / (name & ".nimble")):
      return candidate
  raise newException(CatchableError, name & " unresolved - run `nimble setup`")

proc restoreLeopardTestSources() =
  ## nim-leopard's CMakeLists.txt declares test executables from `tests/`, which
  ## Nimble strips on install, so cmake fails before building libleopard. Same
  ## workaround as logos-delivery's Leopard.mk, which only covers its own build.
  ## Every installed revision is patched: the dependency solve and
  ## logos-delivery's own task can pick different ones.
  let (output, _) = gorgeEx("nimble path leopard")
  for line in output.splitLines():
    let pkgDir = line.strip()
    if not pkgDir.isAbsolute() or not dirExists(pkgDir / "vendor" / "leopard"):
      continue
    let testsDir = pkgDir / "vendor" / "leopard" / "tests"
    mkDir testsDir
    for stub in ["benchmark.cpp", "experiments.cpp"]:
      if not fileExists(testsDir / stub):
        writeFile(testsDir / stub, "int main() { return 0; }\n")

proc leopardCmakeParams(): string =
  ## Mirrors the Leopard-RS flags of logos-delivery's portable config.nims, which
  ## is not installed with the package: position-independent, no -march=native,
  ## and the SIMD baseline it needs instead. Windows keeps nim-leopard's own flags.
  if getEnv("NIM_PARAMS").contains("LeopardCmakeFlags"):
    return ""
  when defined(windows):
    return ""
  else:
    let base =
      when defined(macosx): "-DCMAKE_BUILD_TYPE=Release -DENABLE_OPENMP=off"
      else: "-DCMAKE_BUILD_TYPE=Release"
    let cxxFlags =
      when defined(amd64) or defined(i386):
        when defined(macosx): " -DCMAKE_CXX_FLAGS=-march=haswell"
        else: " -DCMAKE_CXX_FLAGS=-mssse3"
      else: ""
    return " -d:\"LeopardCmakeFlags=" & base &
      " -DCMAKE_POSITION_INDEPENDENT_CODE=ON -DCOMPILER_SUPPORTS_MARCH_NATIVE=FALSE" &
      cxxFlags & "\""

### Tasks

task liblogosdelivery, "Build the liblogosdelivery these bindings link against":
  ## Delegates to logos-delivery's own build task.
  ## Consumers set NIM_PARAMS (e.g. -d:disable_rln) and LIBLOGOSDELIVERY_OUT;
  ## neither is decided here.
  restoreLeopardTestSources()
  putEnv("NIM_PARAMS", getEnv("NIM_PARAMS") & leopardCmakeParams())
  let pkgDir = nimblePkgDir("logos_delivery")
  withDir pkgDir:
    exec "nimble liblogosdelivery"

  let outDir = getEnv("LIBLOGOSDELIVERY_OUT")
  if outDir.len > 0:
    let lib = DynlibFormat % "logosdelivery"
    mkDir outDir
    cpFile pkgDir / "build" / lib, outDir / lib

proc runMobileTask(taskName, built: string) =
  ## Delegates to logos-delivery's mobile task, then copies `built`, relative to
  ## its build/ directory, to LIBLOGOSDELIVERY_OUT. config.nims, installed with
  ## logos-delivery, supplies the cross-compile flags, so leopardCmakeParams is
  ## not added. nat_traversal's root goes on NIM_PARAMS for the task to find.
  restoreLeopardTestSources()
  putEnv("NIM_PARAMS", getEnv("NIM_PARAMS") &
    " --path:\"" & nimblePkgDir("nat_traversal") & "\"")
  let pkgDir = nimblePkgDir("logos_delivery")
  withDir pkgDir:
    exec "nimble " & taskName

  let outDir = getEnv("LIBLOGOSDELIVERY_OUT")
  if outDir.len > 0:
    mkDir outDir
    cpFile pkgDir / "build" / built, outDir / extractFilename(built)

task liblogosdeliveryAndroid, "Build liblogosdelivery for Android":
  ## Needs ANDROID_NDK_ROOT, CPU (arm64, amd64, i386, arm) and ABIDIR.
  runMobileTask("libLogosDeliveryAndroid",
    "android" / getEnv("ABIDIR") / "liblogosdelivery.so")

task liblogosdeliveryIOS, "Build liblogosdelivery for iOS":
  ## Needs IOS_SDK (iphoneos, iphonesimulator), IOS_ARCH and IOS_SDK_PATH.
  runMobileTask("libLogosDeliveryIOS",
    "ios" / (getEnv("IOS_SDK") & "-" & getEnv("IOS_ARCH")) / "liblogosdelivery.a")
