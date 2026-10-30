//go:build windows && arm64 && cgo && duckdb_use_static_lib

#include <cstdlib>

// _atoi64 is declared with _CRTIMP, unlike the standard strtol declaration.
extern "C" long long parse_cpp(const char *value) {
    return _atoi64(value);
}
