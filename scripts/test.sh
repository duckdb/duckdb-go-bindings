#!/usr/bin/env bash
# Run from the repository root. Make forwards caller-supplied environment and
# command-line settings, including TAGS, CC, AR, BASE_URL and LIB_WORK.
set -euo pipefail

usage() {
	echo "usage: $0 [prebuilt|packaged|static|dynamic]" >&2
	exit 2
}
[[ $# -le 1 && ${1:-prebuilt} =~ ^(prebuilt|packaged|static|dynamic)$ ]] || usage
mode=${1:-prebuilt}

brood() {
	GOWORK=off go run ./internal/cmd/brood "$@"
}

# Keep the assignment separate from eval so a brood failure stops the script.
lib_env=$(brood env)
eval "$lib_env"
tags=${TAGS:-}

case "$mode" in
	packaged)
		export GOWORK=off GOPROXY="$MODULE_PROXY" GOMODCACHE="$MODULE_CACHE"
		;;
	static)
		static_env=$(./scripts/static-env.sh)
		eval "$static_env"
		tags="${tags:+$tags,}duckdb_use_static_lib"
		;;
	dynamic)
		brood fetch-shared
		export CGO_ENABLED=1 CGO_LDFLAGS="\"-L$DYNAMIC_DIR\""
		case "$PLATFORM" in
			windows-*)
				dynamic_path=$(cd "$DYNAMIC_DIR" && pwd)
				export PATH="$dynamic_path:$PATH"
				;;
			# An rpath, since macOS SIP strips DYLD_* from child processes.
			*) export CGO_LDFLAGS="$CGO_LDFLAGS \"-Wl,-rpath,$DYNAMIC_DIR\"" ;;
		esac
		tags="${tags:+$tags,}duckdb_use_lib"
		;;
esac

go test -mod=readonly -count=1 -tags="$tags" ./...
# Only the bindings depend on the tags; brood's tests ran above.
go test -mod=readonly -count=1 -tags="${tags:+$tags,}debug_bindings,duckdb_arrow" .
case "$mode" in
	prebuilt|packaged) go mod verify ;;
esac
