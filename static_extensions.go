//go:build !duckdb_use_lib

package duckdb_go_bindings

/*
#include <stdint.h>

// Defined by the static extension loader for the extensions linked into the
// program: static_extension_loader.c for the prebuilt libraries, or the one
// DuckDB generates (make static_extension_loader) with duckdb_use_static_lib.
int32_t duckdb_register_static_extensions(void);
*/
import "C"

func init() {
	// Statically linked extensions must be registered before a database is
	// opened. A failed registration makes opening any database fail with the
	// reason, so there is nothing to report here.
	C.duckdb_register_static_extensions()
}
