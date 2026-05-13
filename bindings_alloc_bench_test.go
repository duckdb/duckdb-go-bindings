package duckdb_go_bindings

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateVarcharEmpty(t *testing.T) {
	defer VerifyAllocationCounters()
	v := CreateVarchar("")
	defer DestroyValue(&v)
	require.NotNil(t, v.Ptr)
}

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
// Run with: CGO_ENABLED=1 go test -bench . -benchmem -benchtime=500ms ./...
//
// Reading allocs/op: this column mostly reflects Go heap allocations only.
// Pure C allocations (libc malloc/C.CString paths that do not touch the GC scan) may under-report allocs/op
// despite visible CPU — also look at ns/op and malloc profiling (jemalloc_stats, Tracy, perf).
//
// Non-empty SQL/query strings use withNULString: pooled []byte + copy instead of C.CString.
// Large stack buffers handed to cgo still escape to the Go heap (so we rely on pooling, not big stack arrays).
//
// CreateEnumType with many labels: allocNames uses two duckdb_malloc calls (pointer array + contiguous NUL-string blob);
// see BenchmarkCreateEnumType_64Names.
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

func BenchmarkPrepare_ShortSQL(b *testing.B) {
	_, conn := benchMustOpen(b)

	const sql = "SELECT $1::VARCHAR"
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var stmt PreparedStatement
		if Prepare(conn, sql, &stmt) != StateSuccess {
			b.Fatal(PrepareError(stmt))
		}
		DestroyPrepare(&stmt)
	}
}

// BenchmarkQuery_simpleSelect calls duckdb_query in a loop; SQL text passes through withNULString (pool/copy).
func BenchmarkQuery_simpleSelect(b *testing.B) {
	_, conn := benchMustOpen(b)
	const sql = `SELECT 1 AS x`
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var res Result
		if Query(conn, sql, &res) != StateSuccess {
			b.Fatal("query")
		}
		DestroyResult(&res)
	}
}

func BenchmarkBindVarchar_preparedHotPath(b *testing.B) {
	_, conn := benchMustOpen(b)
	var stmt PreparedStatement
	requirePrepare := func() {
		if Prepare(conn, "SELECT $1::VARCHAR WHERE $1 IS NOT NULL", &stmt) != StateSuccess {
			b.Fatal(PrepareError(stmt))
		}
	}
	requirePrepare()
	b.Cleanup(func() { DestroyPrepare(&stmt) })

	s := "ingest-tag-value-pair-short"
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if BindVarchar(stmt, 1, s) != StateSuccess {
			b.Fatal("bind varchar")
		}
		ClearBindings(stmt)
	}
}

func BenchmarkBindBlob_preparedHotPath(b *testing.B) {
	_, conn := benchMustOpen(b)
	var stmt PreparedStatement
	if Prepare(conn, "SELECT $1", &stmt) != StateSuccess {
		b.Fatal("prepare select $1 blob")
	}
	b.Cleanup(func() { DestroyPrepare(&stmt) })

	blob := make([]byte, 128)
	for j := range blob {
		blob[j] = byte(j)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if BindBlob(stmt, 1, blob) != StateSuccess {
			b.Fatal("bind blob")
		}
		ClearBindings(stmt)
	}
}

func BenchmarkCreateVarchar_destroy(b *testing.B) {
	const s = "hello-duckdb-go-bindings"
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v := CreateVarchar(s)
		DestroyValue(&v)
	}
}

func BenchmarkCreateVarcharLength_destroy(b *testing.B) {
	const s = "varchar-length-variant"
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v := CreateVarcharLength(s, IdxT(len(s)))
		DestroyValue(&v)
	}
}

func BenchmarkValueToString_int(b *testing.B) {
	v := CreateInt64(42)
	defer DestroyValue(&v)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValueToString(v)
	}
}

func BenchmarkValueToString_fromVarchar(b *testing.B) {
	const s = "round-trip-text"
	v := CreateVarchar(s)
	defer DestroyValue(&v)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValueToString(v)
	}
}

func BenchmarkValidUtf8Check_512B(b *testing.B) {
	buf := make([]byte, 512)
	for i := range buf {
		buf[i] = 'a'
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ed := ValidUtf8Check(buf)
		DestroyErrorData(&ed)
	}
}

func BenchmarkNewBigNum_32B_destroy(b *testing.B) {
	data := make([]byte, 32)
	for i := range data {
		data[i] = byte(i + 1)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bn := NewBigNum(data, false)
		DestroyBigNum(&bn)
	}
}

// CreateEnumType with many labels does two duckdb_malloc calls (pointer array + string blob) via allocNames
// instead of N separate C strings; see BenchmarkCreateEnumType_64Names.
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
