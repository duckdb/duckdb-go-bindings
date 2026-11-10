package duckdb_go_bindings

import (
	"fmt"
	"testing"
	"unsafe"

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

func TestCreateVarcharEmpty(t *testing.T) {
	defer VerifyAllocationCounters()
	v := CreateVarchar("")
	defer DestroyValue(&v)
	require.NotNil(t, v.Ptr)
}

// benchMustOpen allocates an in-memory DB and connection for micro-benchmarks.
func benchMustOpen(b *testing.B) (Database, Connection) {
	b.Helper()

	var db Database
	var config Config
	var errMsg string
	if OpenExt(":memory:", &db, config, &errMsg) != StateSuccess {
		b.Fatal("failed to open in-memory DB")
	}
	b.Cleanup(func() { Close(&db) })

	var conn Connection
	if Connect(db, &conn) != StateSuccess {
		Close(&db)
		b.Fatal("failed to open connection")
	}
	b.Cleanup(func() { Disconnect(&conn) })

	return db, conn
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

	const s = "ingest-tag-value-pair-short"
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if BindVarchar(stmt, 1, s) != StateSuccess {
			b.Fatal("failed to bind VARCHAR")
		}
		ClearBindings(stmt)
	}
}

func BenchmarkBindBlob_preparedHotPath(b *testing.B) {
	_, conn := benchMustOpen(b)
	var stmt PreparedStatement
	if Prepare(conn, "SELECT $1", &stmt) != StateSuccess {
		b.Fatal(PrepareError(stmt))
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
			b.Fatal("failed to bind BLOB")
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

func BenchmarkCreateErrorData_destroy(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ed := CreateErrorData(ErrorTypeInvalidInput, "invalid input")
		DestroyErrorData(&ed)
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

func BenchmarkAllocNames(b *testing.B) {
	for _, n := range []int{1, 8, 64} {
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			names := make([]string, n)
			for i := range names {
				names[i] = fmt.Sprintf("field_%02d", i)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				freeNameList(allocNames(names))
			}
		})
	}
}

func BenchmarkAllocLogicalTypes(b *testing.B) {
	typ := CreateLogicalType(TypeInteger)
	defer DestroyLogicalType(&typ)
	benchmarkAllocHandles(b, typ, allocLogicalTypes)
}

func BenchmarkAllocValues(b *testing.B) {
	v := CreateInt32(42)
	defer DestroyValue(&v)
	benchmarkAllocHandles(b, v, allocValues)
}

func benchmarkAllocHandles[T, P any](b *testing.B, handle T, alloc func([]T) *P) {
	b.Helper()
	for _, n := range []int{1, 8, 64} {
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			handles := make([]T, n)
			for i := range handles {
				handles[i] = handle
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				Free(unsafe.Pointer(alloc(handles)))
			}
		})
	}
}

func BenchmarkCreateStructType_64Fields(b *testing.B) {
	typ := CreateLogicalType(TypeInteger)
	defer DestroyLogicalType(&typ)
	types := make([]LogicalType, 64)
	names := make([]string, 64)
	for i := range types {
		types[i] = typ
		names[i] = fmt.Sprintf("field_%02d", i)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := CreateStructType(types, names)
		DestroyLogicalType(&result)
	}
}

func BenchmarkCreateListValue_64Items(b *testing.B) {
	typ := CreateLogicalType(TypeInteger)
	defer DestroyLogicalType(&typ)
	v := CreateInt32(42)
	defer DestroyValue(&v)
	values := make([]Value, 64)
	for i := range values {
		values[i] = v
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := CreateListValue(typ, values)
		DestroyValue(&result)
	}
}
