//go:build windows && arm64 && cgo && duckdb_use_static_lib

package cpp

import (
	"testing"

	_ "github.com/duckdb/duckdb-go-bindings"
)

// Exercise a C++ CRT call alongside the bindings, not just their C bridge.
func TestStaticCRT(t *testing.T) {
	if got := parse("-4294967319"); got != -4294967319 {
		t.Fatalf("parse(-4294967319) = %d, want -4294967319", got)
	}
}
