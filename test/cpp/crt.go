//go:build windows && arm64 && cgo && duckdb_use_static_lib

package cpp

/*
#include <stdlib.h>
long long parse_cpp(const char *value);
*/
import "C"

import "unsafe"

func parse(value string) int64 {
	text := C.CString(value)
	defer C.free(unsafe.Pointer(text))
	return int64(C.parse_cpp(text))
}
