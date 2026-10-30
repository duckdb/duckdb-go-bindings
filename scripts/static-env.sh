#!/usr/bin/env bash
# Run from the repository root. Stage static libraries and print shell exports
# for building Go applications with duckdb_use_static_lib.
set -euo pipefail

brood() {
	GOWORK=off go run ./internal/cmd/brood "$@"
}

lib_env=$(brood env)
eval "$lib_env"

if [[ $PLATFORM == windows-arm64 ]]; then
	: "${MSVC_LIB_DIR:?Set MSVC_LIB_DIR to the VS 2026 ARM64 CRT library directory}"
	: "${UCRT_LIB_DIR:?Set UCRT_LIB_DIR to the Windows SDK ARM64 UCRT library directory}"
	: "${WINDOWS_LIB_DIR:?Set WINDOWS_LIB_DIR to the Windows SDK ARM64 UM library directory}"
	# These settings must apply to runtime/cgo and every C/C++ dependency,
	# not just to the bindings' platform module.
	export CGO_CPPFLAGS="${CGO_CPPFLAGS:+$CGO_CPPFLAGS }-D_CRTIMP="
	AR=${AR:-llvm-ar}
	# Resolve MinGW's own libraries before the MSVC search paths.
	# CC may include command arguments.
	# shellcheck disable=SC2086
	compiler_lib=$(${CC:-cc} -print-file-name=libmingw32.a)
	compiler_lib=${compiler_lib//\\//}
	if [[ ! -f $compiler_lib ]]; then
		echo "Windows ARM64 static linking requires an ARM64 llvm-mingw compiler (set CC)" >&2
		exit 1
	fi
	export CGO_LDFLAGS="\"-L${compiler_lib%/*}\" \"-L$MSVC_LIB_DIR\" \"-L$UCRT_LIB_DIR\" \"-L$WINDOWS_LIB_DIR\" ${CGO_LDFLAGS:-}"
fi

brood fetch
# An archive tolerates cgo repeating linker flags across packages.
# CC and AR may include command arguments.
# shellcheck disable=SC2086
${CC:-cc} -c "$NATIVE_DIR/static_extension_loader.c" -o "$NATIVE_DIR/loader.o"
# shellcheck disable=SC2086
${AR:-ar} rcs "$NATIVE_DIR/libduckdb_static_extension_loader.a" "$NATIVE_DIR/loader.o"

printf 'export CGO_ENABLED=1\n'
printf 'export CGO_LDFLAGS=%q\n' "-lduckdb_static_extension_loader $STATIC_LDFLAGS ${CGO_LDFLAGS:-}"
if [[ ${CGO_CPPFLAGS+x} ]]; then
	printf 'export CGO_CPPFLAGS=%q\n' "$CGO_CPPFLAGS"
fi
