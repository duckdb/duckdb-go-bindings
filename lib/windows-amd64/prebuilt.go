//go:build windows && amd64

package duckdb_go_bindings_platform

/*
#cgo CPPFLAGS: -I${SRCDIR} -DDUCKDB_STATIC_BUILD
#cgo LDFLAGS: -lduckdb_httplib -lduckdb_loadable_extensions -lautocomplete_extension -lcore_functions_extension -licu_extension -ljson_extension -lparquet_extension -ltpcds_extension -ltpch_extension -Wl,--start-group -lduckdb_static_part1 -lduckdb_static_part2 -Wl,--end-group -lws2_32 -lwsock32 -lrstrtmgr -lstdc++ -lm --static -L${SRCDIR}
#include <duckdb.h>
*/
import "C"
