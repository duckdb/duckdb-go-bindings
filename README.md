# duckdb-go-bindings

![Tests status](https://github.com/duckdb/duckdb-go-bindings/actions/workflows/run_tests.yml/badge.svg)

This repository wraps DuckDB's C API calls in Go native types and functions.

Minimum Go version: 1.24.

#### 🚧 WORK IN PROGRESS 🚧

> [!IMPORTANT]
> Some type aliases and function wrappers are still missing.
>
> Breaking changes can happen.

## Releases

### Semantic versioning

Starting with DuckDB `v1.5.0` the duckdb-go-bindings package changes its versioning scheme to contain
- the DuckDB release version
- the duckdb-go-bindings patch iteration

For `v1.5.0` the corresponding tagged release is `v0.10500.0`.

### Single module (v0.3.0+)

Starting with v0.3.0, the module includes pre-built static libraries for all platforms. Simply import `github.com/duckdb/duckdb-go-bindings`.

| duckdb version | module version |
| -------------- | -------------- |
| v1.4.5         | v0.3.5         |
| v1.4.4         | v0.3.4         |
| v1.4.3         | v0.3.0         |

### Legacy per-platform submodules (v0.1.x)

Older versions require platform-specific imports (e.g., `github.com/duckdb/duckdb-go-bindings/darwin-arm64`). These tags continue to work.

| duckdb version | main    | darwin  | linux   | windows |
| -------------- | ------- | ------- | ------- | ------- |
| v1.4.3         | v0.1.24 | v0.1.24 | v0.1.24 | v0.1.24 |
| v1.4.2         | v0.1.23 | v0.1.23 | v0.1.23 | v0.1.23 |
| v1.4.1         | v0.1.21 | v0.1.21 | v0.1.21 | v0.1.21 |
| v1.4.0         | v0.1.19 | v0.1.19 | v0.1.19 | v0.1.19 |
| v1.3.2         | v0.1.17 | v0.1.12 | v0.1.12 | v0.1.12 |
| v1.3.1         | v0.1.16 | v0.1.11 | v0.1.11 | v0.1.11 |
| v1.3.0         | v0.1.15 | v0.1.10 | v0.1.10 | v0.1.10 |
| v1.2.2         | v0.1.14 | v0.1.9  | v0.1.9  | v0.1.9  |
| v1.2.1         | v0.1.13 | v0.1.8  | v0.1.8  | v0.1.8  |
| v1.2.0         | v0.1.10 | v0.1.5  | v0.1.5  | v0.1.5  |

## Installation

Simply import the module in your Go project:

```go
import "github.com/duckdb/duckdb-go-bindings"
```

The module includes pre-built static libraries for all supported platforms:

- darwin-amd64
- darwin-arm64
- linux-amd64
- linux-arm64
- windows-amd64

Platform detection and linking is handled automatically through cgo directives. `CGO_ENABLED=1` is required, and your system needs a C compiler.

## Upgrading to DuckDB v2

The Go API is unchanged: the bindings keep using DuckDB's C API v1, which DuckDB v2 still ships. What else changes depends on how you link DuckDB.

**Pre-built static libraries (default):** nothing to do. DuckDB v2 no longer loads statically linked extensions on its own; the bindings register them when the package is initialized, before any database can be opened. The same extensions as before are loaded, and installing and loading other extensions works as before.

**Dynamic libraries (`duckdb_use_lib`):** nothing to do. The shared library registers its own extensions. DuckDB v2 publishes it as `duckdb-shared-libs-<platform>.tar.gz`, or use DuckDB's install script (see [Dynamic Libraries](#alternative-dynamic-libraries)).

**Custom static libraries (`duckdb_use_static_lib`):** your build needs changes (see [Custom Static Libraries](#alternative-custom-static-libraries)):

- Link a static extension loader that defines `duckdb_register_static_extensions()`, which the bindings call before any database is opened. Without it, the build fails with an undefined symbol.
- Link the new set of archives: the third-party libraries are now part of `libduckdb_static.a`, `libduckdb_generated_extension_loader.a` and DuckDB's `bundle-library` target are gone, and `libduckdb_httplib.a` and `libduckdb_loadable_extensions.a` are new. Put `libduckdb_static.a` last.
- If your build already registers its extensions before `main` (`static_extension_autoregister.cpp`, linked with `-Wl,-u,duckdb_register_static_extensions`), it keeps working, since registering an extension twice is a no-op. The bindings now call the loader themselves, so neither is needed anymore.

## Local Development

To develop locally, copy the workspace template file:

```bash
cp go.work.dev go.work
```

This sets up Go workspaces to use the local lib/\* submodules instead of fetching from the module proxy.

### Static library packaging

GitHub blocks individual files larger than 100 MiB. The fetch workflow runs
`make package.static.libs PLATFORM=<platform>` after downloading each platform's
archives to keep the prebuilt libraries below that limit. Packaging needs Python
3.9 or newer and LLVM 19 or newer (`llvm-ar`, `llvm-nm`, and `ld.lld`); consumers
only need their usual C compiler.
The tool paths can be overridden with `LLVM_AR`, `LLVM_NM`, and `LD_LLD`.

Oversized Linux ELF objects are relinked with merged sections, preserving every
exported symbol, including symbols unused by the bindings or bundled extensions.
No section garbage collection or extension-specific symbol list is used.
Oversized archives containing multiple objects are split at object boundaries.
Their original `.a` filename becomes a GNU linker `GROUP` script that rescans
the parts, so `-lduckdb_static` continues to work on Linux and MinGW. Keep all
the part archives alongside the script when copying packaged libraries.

Archives that already fit, including the current macOS libraries, are unchanged.
Packaging fails if an individual object still exceeds the limit or an oversized
archive uses an unsupported platform. `make fetch.static.libs` alone keeps the
upstream archives intact for nightly artifact validation.

## Nightly artifact validation

The `Nightly` workflow accepts a full DuckDB commit SHA and downloads the
matching Linux, macOS, and Windows artifacts from DuckDB's staging endpoint. It
compiles the bindings against the artifact header, summarizes the resulting
header diff, exercises both static and dynamic linking, verifies the linked
source ID, and installs and loads `httpfs` from a clean extension directory.

## Releasing a new DuckDB version

Prepare normal releases from `main`. For LTS releases that stay on DuckDB 1.4 Andium, use the `v1.4-andium` branch.

1. Create a new branch and update the `DUCKDB_VERSION` in the `Makefile`.
2. Invoke the `Fetch and Push Libs` workflow on the new branch (it commits fetched libs to the branch; it does not tag).
3. If the header (`duckdb.h`) has changes, add all changes (new types, functions, etc.) to the bindings.
4. Open a PR.
5. Wait for all tests to pass.
6. Merge the PR into `main` (direct pushes to `main` are not allowed).
7. Publish tags using the release script (re-entrant, safe to run multiple times):

```bash
./scripts/release.sh v0.10502.0           # pushes to 'origin'
./scripts/release.sh v0.10502.0 upstream  # pushes to custom remote
```

The script handles:

- Tagging and pushing lib/\* submodules
- Updating root module deps (stops for PR if changes needed)
- Tagging and pushing root module

Run it again after merging the deps PR to complete the release.

## Build Configuration

### Default: Pre-built Static Libraries

By default (no build tags needed), the module automatically links against the pre-built static libraries for your platform:

```bash
go build  # Just works!
go test   # Just works!
```

The appropriate library and linker flags are selected automatically based on your OS and architecture.

### Alternative: Custom Static Libraries

To use your own DuckDB static libraries instead of the pre-built ones, use the `duckdb_use_static_lib` build tag and provide the libraries via `CGO_LDFLAGS`.

DuckDB's install script fetches the static libraries of a release for the current platform into `~/.duckdb/lib/<version>/release_<platform>`:

```bash
curl https://install.duckdb.org | DUCKDB_INSTALL=static sh
```

Statically linked extensions are only loaded once they are registered with DuckDB. With `duckdb_use_static_lib`, the bindings call `duckdb_register_static_extensions()` before any database is opened, and your build must link it. DuckDB generates it for the extensions and capabilities (`httplib`, `loadable_extensions`) of a build:

```bash
# In your DuckDB checkout, after building it
make static_extension_loader  # writes build/release/static_extension_loader.c
cc -c -Isrc/include build/release/static_extension_loader.c -o static_extension_loader.o
ar rcs libduckdb_static_extension_loader.a static_extension_loader.o
```

Without a DuckDB checkout, use [static_extension_loader.c](static_extension_loader.c) as a starting point, keeping the describe functions for the archives you link. Compile it against the `duckdb_static_extension.h` that comes with the libraries. The loader registers capabilities (`httplib`, `loadable_extensions`) before extensions.

Then link the loader, the extension archives, and `libduckdb_static.a` last:

```bash
EXTENSIONS="-lduckdb_static_extension_loader -lduckdb_httplib -lduckdb_loadable_extensions -ljson_extension"

# Darwin/macOS
CGO_ENABLED=1 \
  CGO_LDFLAGS="$EXTENSIONS -lduckdb_static -lc++ -L/path/to/lib" \
  go build -tags=duckdb_use_static_lib

# Linux
CGO_ENABLED=1 \
  CGO_LDFLAGS="$EXTENSIONS -lduckdb_static -lstdc++ -lm -ldl -L/path/to/lib" \
  go build -tags=duckdb_use_static_lib

# Windows
CGO_ENABLED=1 \
  CGO_LDFLAGS="$EXTENSIONS -lduckdb_static -lws2_32 -lwsock32 -lrstrtmgr -lstdc++ -lm --static -L/path/to/lib" \
  go build -tags=duckdb_use_static_lib
```

### Alternative: Dynamic Libraries

To link against a shared DuckDB library, use the `duckdb_use_lib` build tag. `curl https://install.duckdb.org | DUCKDB_INSTALL=shared sh` installs the shared library of a release into `~/.duckdb/lib/<version>/release_<platform>`.

```bash
# Darwin/macOS
CGO_ENABLED=1 \
  CGO_LDFLAGS="-lduckdb -L/path/to/dir" \
  DYLD_LIBRARY_PATH=/path/to/dir \
  go build -tags=duckdb_use_lib

# Linux
CGO_ENABLED=1 \
  CGO_LDFLAGS="-lduckdb -L/path/to/dir" \
  LD_LIBRARY_PATH=/path/to/dir \
  go build -tags=duckdb_use_lib
```

## Arrow functions

Provide the duckdb_arrow build tag if you want to use arrow functions
