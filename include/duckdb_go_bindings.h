#include <duckdb.h>

static void duckdb_go_bindings_set_logical_type(duckdb_logical_type *types_ptr, duckdb_logical_type type, idx_t index) {
	types_ptr[index] = type;
}

static void duckdb_go_bindings_set_value(duckdb_value *values_ptr, duckdb_value value, idx_t index) {
	values_ptr[index] = value;
}
