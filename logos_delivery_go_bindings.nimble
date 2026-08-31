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
requires "https://github.com/logos-messaging/logos-delivery#53d42f3f"

### Helpers

proc nimblePkgDir(name: string): string =
  ## `nimble path` reports where a dependency was installed. It prints a banner
  ## on stdout and exits 0 even when the package is missing, so take the last
  ## line and check it.
  let (output, _) = gorgeEx("nimble path " & name)
  result = output.strip().splitLines()[^1].strip()
  if not result.isAbsolute() or not dirExists(result):
    raise newException(CatchableError, name & " unresolved - run `nimble setup`")

### Tasks

task liblogosdelivery, "Build the liblogosdelivery these bindings link against":
  ## Delegates to logos-delivery's own build task.
  ## Consumers set NIM_PARAMS (e.g. -d:disable_rln) and LIBLOGOSDELIVERY_OUT;
  ## neither is decided here.
  let pkgDir = nimblePkgDir("logos_delivery")
  withDir pkgDir:
    exec "nimble liblogosdelivery"

  let outDir = getEnv("LIBLOGOSDELIVERY_OUT")
  if outDir.len > 0:
    let lib = DynlibFormat % "logosdelivery"
    mkDir outDir
    cpFile pkgDir / "build" / lib, outDir / lib
