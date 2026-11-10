package duckdb_go_bindings

import (
	"os"
	"strings"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/require"
)

func TestStringTData(t *testing.T) {
	defer VerifyAllocationCounters()
	varcharT := CreateLogicalType(TypeVarchar)
	defer DestroyLogicalType(&varcharT)
	chunk := CreateDataChunk([]LogicalType{varcharT})
	defer DestroyDataChunk(&chunk)
	vec := DataChunkGetVector(chunk, 0)

	for _, tc := range []struct {
		name  string
		value string
	}{
		{"empty", ""},
		{"inlined", "short"},
		{"inlined_boundary", strings.Repeat("a", 12)},
		{"external_boundary", strings.Repeat("b", 13)},
		{"external", strings.Repeat("c", 512)},
		{"unicode", "Grüße aus Berlin"},
		{"embedded_NUL", "a\x00b\x00c"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			VectorAssignStringElement(vec, 0, tc.value)
			strT := (*StringT)(VectorGetData(vec))
			require.Equal(t, uint32(len(tc.value)), StringTLength(*strT))
			got := StringTData(strT)
			require.Equal(t, tc.value, got)

			// The returned Go string must survive changes to DuckDB's storage.
			if StringIsInlined(*strT) {
				VectorAssignStringElement(vec, 0, strings.Repeat("x", len(tc.value)))
			} else {
				// Reassignment leaves the old external payload intact. Use the
				// public C layout to overwrite those bytes while they are alive.
				external := (*struct {
					length uint32
					prefix [4]byte
					ptr    *byte
				})(unsafe.Pointer(strT))
				clear(unsafe.Slice(external.ptr, len(tc.value)))
			}
			require.Equal(t, tc.value, got)
		})
	}
}

func TestStringTDataLarge(t *testing.T) {
	if os.Getenv("DUCKDB_TEST_LARGE_STRINGS") != "1" {
		t.Skip("Set DUCKDB_TEST_LARGE_STRINGS=1 to test a 2 GiB string (requires several GiB of memory)")
	}

	conn := openTestConnection(t)
	var result Result
	defer DestroyResult(&result)
	require.Equal(t, StateSuccess, Query(conn, "SELECT repeat('a', 2147483648)", &result), ResultError(&result))
	chunk := FetchChunk(result)
	defer DestroyDataChunk(&chunk)
	vec := DataChunkGetVector(chunk, 0)
	strT := (*StringT)(VectorGetData(vec))
	require.Equal(t, uint32(1<<31), StringTLength(*strT))

	got := StringTData(strT)
	require.Equal(t, uint64(1<<31), uint64(len(got)))
	require.Equal(t, len(got), strings.Count(got, "a"))
}

var benchString string

func BenchmarkStringTData(b *testing.B) {
	for _, s := range []string{"short", "a longer string stored outside the string_t"} {
		b.Run(s, func(b *testing.B) {
			varcharT := CreateLogicalType(TypeVarchar)
			defer DestroyLogicalType(&varcharT)
			chunk := CreateDataChunk([]LogicalType{varcharT})
			defer DestroyDataChunk(&chunk)
			vec := DataChunkGetVector(chunk, 0)
			VectorAssignStringElement(vec, 0, s)
			strT := (*StringT)(VectorGetData(vec))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				benchString = StringTData(strT)
			}
		})
	}
}
