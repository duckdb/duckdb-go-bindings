//go:build !duckdb_use_lib

package duckdb_go_bindings

import "testing"

// TestStaticOnlyExtensions checks the extensions that only the static loader
// registers; the shared library doesn't include them.
func TestStaticOnlyExtensions(t *testing.T) {
	requireBuiltinExtensions(t, openTestConnection(t), "tpcds", "tpch")
}
