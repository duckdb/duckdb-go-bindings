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

Starting with v0.3.0, the module selects pre-built static libraries for all platforms. Simply import `github.com/duckdb/duckdb-go-bindings`.

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

The module selects pre-built static libraries for all supported platforms:

- darwin-amd64
- darwin-arm64
- linux-amd64
- linux-arm64
- windows-amd64

Platform detection and linking is handled automatically through cgo directives. `CGO_ENABLED=1` is required, and your system needs a C compiler.

## Upgrading to DuckDB v2

The bindings keep using DuckDB's C API v1, which DuckDB v2 still ships. The Go
API adds `ErrorTypeResourceInUse`, `ErrorTypeDataCorruption`, and nanosecond
timestamps with time zones (`TypeTimestampTZNS`, `CreateTimestampTZNS`, and
`GetTimestampTZNS`). Existing calls keep their signatures. What else changes
depends on how you link DuckDB.

**Pre-built static libraries (default):** nothing to do. DuckDB v2 no longer
loads statically linked extensions on its own; the bindings register them when
the package is initialized, before any database can be opened. The same
extensions as before are loaded, and installing and loading other extensions
works as before. The libraries now come from the origin in
`duckdb-release.json` instead of GitHub, so builds that restrict module
downloads must allow it.

**Dynamic libraries (`duckdb_use_lib`):** nothing to do. The shared library
registers its own extensions. DuckDB v2 publishes it as
`duckdb-shared-libs-<platform>.tar.gz`, or use DuckDB's install script (see
[Dynamic Libraries](#alternative-dynamic-libraries)).

**Custom static libraries (`duckdb_use_static_lib`):** your build needs changes
(see [Custom Static Libraries](#alternative-custom-static-libraries)):

- Link a static extension loader that defines
  `duckdb_register_static_extensions()`, which the bindings call before any
  database is opened. Without it, the build fails with an undefined symbol.
- Link the new set of archives: the third-party libraries are now part of
  `libduckdb_static.a`, `libduckdb_generated_extension_loader.a` and DuckDB's
  `bundle-library` target are gone, and `libduckdb_httplib.a` and
  `libduckdb_loadable_extensions.a` are new. Put `libduckdb_static.a` last.
- If your build already registers its extensions before `main`
  (`static_extension_autoregister.cpp`, linked with
  `-Wl,-u,duckdb_register_static_extensions`), it keeps working, since
  registering an extension twice is a no-op. The bindings now call the loader
  themselves, so neither is needed anymore.

## Testing

With Go, a C/C++ compiler and Make installed, run:

```bash
make test
```

This uses the native module proxy configured in `duckdb-release.json`, checks
the linked DuckDB version and source ID, runs the default and debug/Arrow tests,
and verifies the module cache. No bucket credentials are required.

The test targets run `scripts/test.sh`, which also runs directly from the
repository root with `prebuilt` (the default), `static`, `dynamic` or
`packaged` (used by `make update.libs`).

## Updating native libraries

The native libraries are published as one Go module per platform, outside this
repository: their static archives exceed GitHub's file size limit.
`duckdb-release.json` pins the DuckDB build, native module version, upstream
tarball checksums and publication origin.

Only a new DuckDB build needs this process. Releases with code changes only bump
the root module's patch version, e.g. to `v0.20000.1`, and keep the native
module pins.

To update, change the pins on a branch, then update either in GitHub Actions or
locally. Either way, open a pull request for the branch as usual, and tag the
root module after merge.

### In GitHub Actions

Run the **Update Native Libraries** workflow on the branch. The branch must be
in this repository: the workflow uploads with the repository's bucket secrets,
which GitHub doesn't pass to forks. It:

1. packages all five native modules and tests them before publishing anything,
2. pushes the updated header, platform imports and `go.mod`/`go.sum` to the
   branch as a "Pin native modules" commit,
3. uploads the modules and lists the bucket in the job log, and
4. starts the Tests workflow on the pin commit.

The push comes before the upload, so a rejected push publishes nothing, and a
failed test leaves the version unpublished for another attempt. These tests only
link the runner's platform, linux-amd64. The Tests workflow is the first to link
the other platforms, after the upload, so a failure there needs a new native
module version.

The workflow refuses to run in a fork or on the default branch. It needs three
repository secrets for an R2 API token with Object Read & Write permission on
the bucket:

| Secret                               | Value                                           |
| ------------------------------------ | ----------------------------------------------- |
| `DUCKDB_GO_BUCKET_ACCESS_KEY_ID`     | the token's access key ID                       |
| `DUCKDB_GO_BUCKET_SECRET_ACCESS_KEY` | the token's secret access key                   |
| `DUCKDB_GO_BUCKET_ENDPOINT_URL_S3`   | `https://<account ID>.r2.cloudflarestorage.com` |

A bucket in the EU jurisdiction needs the
`https://<account ID>.eu.r2.cloudflarestorage.com` endpoint.

### Locally

With an rclone remote for the bucket, run the same steps yourself:

```bash
make update.libs   # package, pin and test
git commit -am "Pin native modules $(jq -r .version duckdb-release.json)"
LIB_DESTINATION=<remote>:duckdb-go make publish.libs
git push
```

Use a current rclone: older versions silently upload without the bucket's
protection against overwriting published files.

### Rules for both

Use the project's Go version for publishing, and a new native module version
whenever previously published bytes change, also after a partial upload:
packaging refuses versions that already exist at the origin. Update the bindings
for any C API changes.

Downloads and packaged modules stay in `LIB_WORK`, outside the checkout
(default: `duckdb-go-libraries` in the system's temporary directory).
`go run ./internal/cmd/brood env` shows the resolved paths. For DuckDB 1.4 LTS,
use the `v1.4-andium` branch's release process.

## Nightly artifact validation

The `Nightly` workflow accepts a full DuckDB commit SHA and downloads matching
v2 staging artifacts. It compiles against their header, shows the header diff,
exercises static and dynamic linking, verifies the linked source ID, and loads
`httpfs` from a clean extension directory. It uses an external scratch directory
and a local workspace replacement, without publishing anything.

The same targets work locally for an alternate build:

```bash
export BASE_URL="https://duckdb-staging.duckdb.org/<sha-prefix>/<version>/duckdb/duckdb/github_release"
export DUCKDB_VERSION="<version>" DUCKDB_SHA="<full-sha>"
make use.libs
make test TAGS=duckdb_nightly
make test.dynamic TAGS=duckdb_nightly
```

`use.libs` installs the build's header and writes an ignored `go.work` selecting
its native module. `BASE_URL` overrides the manifest's upstream URL and drops
its checksum pins, so only the version and source ID supplied through
`DUCKDB_VERSION` and `DUCKDB_SHA` are checked. To return to the pinned build:

```bash
make unuse.libs
unset BASE_URL DUCKDB_VERSION DUCKDB_SHA
```

`unuse.libs` removes the local workspace and restores the pinned header.

## Build Configuration

### Default: Pre-built Static Libraries

By default (no build tags needed), the module automatically links against the pre-built static libraries for your platform. The appropriate library and linker flags are selected automatically based on your OS and architecture.

### Alternative: Custom Static Libraries

To use your own DuckDB v2 static libraries instead of the pre-built ones, use
the `duckdb_use_static_lib` build tag and provide the libraries through
`CGO_LDFLAGS`. DuckDB's install script fetches the static libraries of a release
for the current platform into `~/.duckdb/lib/<version>/release_<platform>`:

```bash
curl https://install.duckdb.org | DUCKDB_INSTALL=static sh
```

Statically linked extensions are only loaded once they are registered with
DuckDB. With `duckdb_use_static_lib`, the bindings call
`duckdb_register_static_extensions()` before any database is opened, and your
build must link it. DuckDB generates it for the extensions and capabilities
(`httplib`, `loadable_extensions`) of a build:

```bash
# In your DuckDB checkout, after building it
make static_extension_loader  # writes build/release/static_extension_loader.c
cc -c -Isrc/include build/release/static_extension_loader.c -o static_extension_loader.o
ar rcs libduckdb_static_extension_loader.a static_extension_loader.o
```

Without a DuckDB checkout, start from the loader of the pre-built libraries:
`go run ./internal/cmd/brood fetch` writes `static_extension_loader.c` to the
native directory reported by `go run ./internal/cmd/brood env`. Keep the
describe functions for the archives you link, and compile it against the
`duckdb_static_extension.h` that comes with the libraries. The loader registers
capabilities before extensions.

Then link the loader, the extension archives, and `libduckdb_static.a` last:

```bash
LIBS="-lduckdb_static_extension_loader -lduckdb_httplib -lduckdb_loadable_extensions -ljson_extension -lduckdb_static"

# Darwin/macOS
CGO_ENABLED=1 \
  CGO_LDFLAGS="$LIBS -lc++ -L/path/to/lib" \
  go build -tags=duckdb_use_static_lib

# Linux
CGO_ENABLED=1 \
  CGO_LDFLAGS="-rdynamic $LIBS -lstdc++ -lm -ldl -L/path/to/lib" \
  go build -tags=duckdb_use_static_lib

# Windows
CGO_ENABLED=1 \
  CGO_LDFLAGS="$LIBS -lws2_32 -lwsock32 -lrstrtmgr -lstdc++ -lm --static -L/path/to/lib" \
  go build -tags=duckdb_use_static_lib
```

To fetch the pinned archives and test custom static linking with their generated
loader on your platform:

```bash
make test.static
```

### Alternative: Dynamic Libraries

To link against a shared DuckDB library, use the `duckdb_use_lib` build tag.
`curl https://install.duckdb.org | DUCKDB_INSTALL=shared sh` installs the shared
library of a release into `~/.duckdb/lib/<version>/release_<platform>`.

```bash
# Darwin/macOS
CGO_ENABLED=1 \
  CGO_LDFLAGS="-lduckdb -L/path/to/dir -Wl,-rpath,/path/to/dir" \
  go build -tags=duckdb_use_lib

# Linux
CGO_ENABLED=1 \
  CGO_LDFLAGS="-lduckdb -L/path/to/dir" \
  LD_LIBRARY_PATH=/path/to/dir \
  go build -tags=duckdb_use_lib
```

To fetch the pinned shared library and run the tests against it on your platform:

```bash
make test.dynamic
```

## Arrow functions

Provide the duckdb_arrow build tag if you want to use arrow functions
