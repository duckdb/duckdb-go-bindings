//go:build !duckdb_use_lib && !duckdb_use_static_lib

// The static extension loader for the archives the prebuilt platform packages
// link, equivalent to the one DuckDB generates (make static_extension_loader).
// httplib and loadable_extensions are capabilities, not extensions, and are
// registered first.

#include <duckdb_static_extension.h>

int32_t duckdb_extension_httplib_describe(duckdb_extension_descriptor *descriptor);
int32_t duckdb_extension_loadable_extensions_describe(duckdb_extension_descriptor *descriptor);
int32_t duckdb_extension_autocomplete_describe(duckdb_extension_descriptor *descriptor);
int32_t duckdb_extension_core_functions_describe(duckdb_extension_descriptor *descriptor);
int32_t duckdb_extension_icu_describe(duckdb_extension_descriptor *descriptor);
int32_t duckdb_extension_json_describe(duckdb_extension_descriptor *descriptor);
int32_t duckdb_extension_parquet_describe(duckdb_extension_descriptor *descriptor);
int32_t duckdb_extension_tpcds_describe(duckdb_extension_descriptor *descriptor);
int32_t duckdb_extension_tpch_describe(duckdb_extension_descriptor *descriptor);

int32_t duckdb_register_static_extensions(void) {
	int32_t result = 0;
	result |= duckdb_register_static_extension(duckdb_extension_httplib_describe) != 0;
	result |= duckdb_register_static_extension(duckdb_extension_loadable_extensions_describe) != 0;
	result |= duckdb_register_static_extension(duckdb_extension_autocomplete_describe) != 0;
	result |= duckdb_register_static_extension(duckdb_extension_core_functions_describe) != 0;
	result |= duckdb_register_static_extension(duckdb_extension_icu_describe) != 0;
	result |= duckdb_register_static_extension(duckdb_extension_json_describe) != 0;
	result |= duckdb_register_static_extension(duckdb_extension_parquet_describe) != 0;
	result |= duckdb_register_static_extension(duckdb_extension_tpcds_describe) != 0;
	result |= duckdb_register_static_extension(duckdb_extension_tpch_describe) != 0;
	return result;
}
