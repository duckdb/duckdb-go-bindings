//go:build !duckdb_use_lib

package duckdb_go_bindings

/*
#cgo CPPFLAGS: -I${SRCDIR}/include -DDUCKDB_STATIC_BUILD

#include <duckdb.h>

// Supplied by the platform module for prebuilt libraries, or by
// DuckDB's generated loader for duckdb_use_static_lib.
int32_t duckdb_register_static_extensions(void);
*/
import "C"

import _ "github.com/duckdb/duckdb-go-bindings/include"

func init() {
	// Register before opening a database. DuckDB reports registration failures
	// when a database is opened.
	C.duckdb_register_static_extensions()
}
