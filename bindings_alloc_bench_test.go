package duckdb_go_bindings

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateEnumType_manyPackedNames(t *testing.T) {
	defer VerifyAllocationCounters()
	names := make([]string, 48)
	for i := range names {
		names[i] = fmt.Sprintf("ev_%03d_alpha", i)
	}
	enumT := CreateEnumType(names)
	defer DestroyLogicalType(&enumT)
	require.NotNil(t, enumT.Ptr)
}

// benchMustOpen allocates an in-memory DB and connection for micro-benchmarks.
func benchMustOpen(b *testing.B) (Database, Connection) {
	b.Helper()
	var db Database
	if Open(":memory:", &db) != StateSuccess {
		b.Fatal("duckdb_open :memory:")
	}
	b.Cleanup(func() { Close(&db) })
	var conn Connection
	if Connect(db, &conn) != StateSuccess {
		Close(&db)
		b.Fatal("duckdb_connect")
	}
	b.Cleanup(func() { Disconnect(&conn) })
	return db, conn
}

// CreateEnumType with many labels: allocNames uses two duckdb_malloc calls (pointer array + contiguous NUL-string blob).
func BenchmarkCreateEnumType_64Names(b *testing.B) {
	names := make([]string, 64)
	for i := range names {
		names[i] = fmt.Sprintf("enum_val_%02d", i)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lt := CreateEnumType(names)
		DestroyLogicalType(&lt)
	}
}
