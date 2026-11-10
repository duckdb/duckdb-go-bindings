package duckdb_go_bindings

import (
	"testing"
	"unsafe"

	"github.com/stretchr/testify/require"
)

func TestValidityMask(t *testing.T) {
	var mask [3]uint64
	ptr := unsafe.Pointer(&mask[0])
	for _, row := range []IdxT{0, 64, 191} {
		ValiditySetRowValid(ptr, row)
	}
	require.Equal(t, [3]uint64{1, 1, 0x8000000000000000}, mask)

	for _, row := range []IdxT{0, 64, 191} {
		require.True(t, ValidityMaskValueIsValid(ptr, row), row)
		require.True(t, ValidityRowIsValid(ptr, row), row)
	}
	for _, row := range []IdxT{1, 31, 63, 65, 127, 128, 190} {
		require.False(t, ValidityMaskValueIsValid(ptr, row), row)
		require.False(t, ValidityRowIsValid(ptr, row), row)
	}

	ValiditySetRowInvalid(ptr, 64)
	require.Equal(t, [3]uint64{1, 0, 0x8000000000000000}, mask)
	ValiditySetRowValidity(ptr, 191, false)
	require.Equal(t, [3]uint64{1, 0, 0}, mask)
	ValiditySetRowValidity(ptr, 127, true)
	require.Equal(t, [3]uint64{1, 0x8000000000000000, 0}, mask)
	ValiditySetRowInvalid(ptr, 0)
	require.Equal(t, [3]uint64{0, 0x8000000000000000, 0}, mask)
}

func TestNilValidityMask(t *testing.T) {
	for _, row := range []IdxT{0, 63, 64, 127, 2047} {
		require.True(t, ValidityRowIsValid(nil, row))
		ValiditySetRowInvalid(nil, row)
		ValiditySetRowValid(nil, row)
		ValiditySetRowValidity(nil, row, false)
		ValiditySetRowValidity(nil, row, true)
	}
}

func TestValidityMaskDataChunk(t *testing.T) {
	conn := openTestConnection(t)
	var create Result
	defer DestroyResult(&create)
	require.Equal(t, StateSuccess, Query(conn, "CREATE TABLE validity_values (v INTEGER)", &create))

	integerT := CreateLogicalType(TypeInteger)
	defer DestroyLogicalType(&integerT)
	chunk := CreateDataChunk([]LogicalType{integerT})
	defer DestroyDataChunk(&chunk)
	vec := DataChunkGetVector(chunk, 0)
	values := unsafe.Slice((*int32)(VectorGetData(vec)), 66)
	for i := range values {
		values[i] = int32(i)
	}
	VectorEnsureValidityWritable(vec)
	mask := VectorGetValidity(vec)
	for _, row := range []IdxT{0, 63, 64, 65} {
		ValiditySetRowInvalid(mask, row)
	}
	ValiditySetRowValidity(mask, 64, true)
	DataChunkSetSize(chunk, 66)

	var appender Appender
	defer AppenderDestroy(&appender)
	require.Equal(t, StateSuccess, AppenderCreateQuery(conn, "INSERT INTO validity_values SELECT * FROM appended_data", []LogicalType{integerT}, "", nil, &appender))
	require.Equal(t, StateSuccess, AppendDataChunk(appender, chunk))
	require.Equal(t, StateSuccess, AppenderFlush(appender))

	var result Result
	defer DestroyResult(&result)
	require.Equal(t, StateSuccess, Query(conn, "SELECT count(*), count(v), sum(v)::BIGINT FROM validity_values", &result))
	require.Equal(t, int64(66), ValueInt64(&result, 0, 0))
	require.Equal(t, int64(63), ValueInt64(&result, 1, 0))
	require.Equal(t, int64(2017), ValueInt64(&result, 2, 0))
}

var benchValid bool

func BenchmarkValidityMaskValueIsValid(b *testing.B) {
	b.Run("stack", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			mask := [1]uint64{uint64(i)}
			benchValid = ValidityMaskValueIsValid(unsafe.Pointer(&mask[0]), 0)
		}
	})
	b.Run("shared", func(b *testing.B) {
		mask := [2]uint64{0x5555555555555555, 0xaaaaaaaaaaaaaaaa}
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			benchValid = ValidityMaskValueIsValid(unsafe.Pointer(&mask[0]), IdxT(i%128))
		}
	})
}

func BenchmarkValidityReadWrite(b *testing.B) {
	var mask [2]uint64
	ptr := unsafe.Pointer(&mask[0])
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		row := IdxT(i % 128)
		ValiditySetRowInvalid(ptr, row)
		benchValid = ValidityRowIsValid(ptr, row)
		ValiditySetRowValid(ptr, row)
		ValiditySetRowValidity(ptr, row, i%2 == 0)
	}
}
