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

The branch must be in this repository, not a fork: the workflow uses the
repository's bucket secrets, which GitHub doesn't pass to forks. Run the
**Update Native Libraries** workflow on the branch. It:

1. packages all five native modules in `$LIB_WORK/modules`,
2. updates the header, platform imports and `go.mod`/`go.sum` to use them, and
   runs `scripts/test.sh packaged` against that local file proxy,
3. pushes those changes to the branch as a "Pin native modules" commit,
4. uploads the modules and lists the destination in the job log, also after a
   failed or partial upload, and
5. starts the Tests workflow on the pin commit.

The push comes before the upload, so a rejected push publishes nothing, and a
failed test leaves the version unpublished for another attempt. These tests only
link the runner's platform, linux-amd64. The Tests workflow is the first to link
the other platforms, after the upload, so a failure there needs a new native
module version.

The workflow refuses to run on the default branch. It installs rclone v1.75.2
with `AnimMouse/setup-rclone` v1.12.0 and defines an S3 remote named `remote`
with `RCLONE_CONFIG_REMOTE_*` environment variables, only for the steps that use
the bucket. Their credentials come from three repository secrets, for an R2 API
token with Object Read & Write permission on the bucket:

| Secret                               | Value                                           |
| ------------------------------------ | ----------------------------------------------- |
| `DUCKDB_GO_BUCKET_ACCESS_KEY_ID`     | the token's access key ID                       |
| `DUCKDB_GO_BUCKET_SECRET_ACCESS_KEY` | the token's secret access key                   |
| `DUCKDB_GO_BUCKET_ENDPOINT_URL_S3`   | `https://<account ID>.r2.cloudflarestorage.com` |

A bucket in the EU jurisdiction needs the
`https://<account ID>.eu.r2.cloudflarestorage.com` endpoint.

### Locally

With a remote for the bucket in your rclone config, run the same steps yourself:

```bash
make update.libs   # package, pin and test
git commit -am "Pin native modules $(jq -r .version duckdb-release.json)"
LIB_DESTINATION=<remote>:duckdb-go make publish.libs
git push
```

`LIB_DESTINATION` takes any rclone remote. Set `RCLONE_CONFIG` to use a config
file other than rclone's default, and `RCLONE` to run an rclone that isn't on
`PATH`. Use rclone v1.75.2 or newer, as the workflow does. Older versions log
`Don't know how to set key "If-None-Match"` and upload without that header,
which turns off the bucket's own refusal to overwrite existing files.

The workflow's remote, as a config section:

```ini
[remote]
type = s3
provider = Cloudflare
access_key_id = <access key ID>
secret_access_key = <secret access key>
endpoint = https://<account ID>.r2.cloudflarestorage.com
region = auto
no_check_bucket = true
```

`no_check_bucket` stops rclone from checking or creating the bucket, which a
token scoped to the bucket can't do.

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

By default (no build tags needed), the module automatically links against the pre-built static libraries for your platform:

```bash
go build  # Just works!
go test   # Just works!
```

The appropriate library and linker flags are selected automatically based on your OS and architecture.

### Alternative: Custom Static Libraries

To use your own DuckDB static library instead of the pre-built ones, use the `duckdb_use_static_lib` build tag and provide library paths via `CGO_LDFLAGS`:

```bash
# Darwin/macOS
CGO_ENABLED=1 \
  CPPFLAGS="-DDUCKDB_STATIC_BUILD" \
  CGO_LDFLAGS="-lduckdb -lc++ -L/path/to/lib" \
  go build -tags=duckdb_use_static_lib

# Linux
CGO_ENABLED=1 \
  CPPFLAGS="-DDUCKDB_STATIC_BUILD" \
  CGO_LDFLAGS="-lduckdb -lstdc++ -lm -ldl -L/path/to/lib" \
  go build -tags=duckdb_use_static_lib

# Windows
CGO_ENABLED=1 \
  CPPFLAGS="-DDUCKDB_STATIC_BUILD" \
  CGO_LDFLAGS="-lduckdb -lws2_32 -lwsock32 -lrstrtmgr -lstdc++ -lm --static -L/path/to/lib" \
  go build -tags=duckdb_use_static_lib
```

### Alternative: Dynamic Libraries

To link against a shared DuckDB library, use the `duckdb_use_lib` build tag:

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
