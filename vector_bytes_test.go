package duckdb_go_bindings

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVectorAssignUTF8Bytes_roundTripChunk(t *testing.T) {
	defer VerifyAllocationCounters()

	varcharT := CreateLogicalType(TypeVarchar)
	defer DestroyLogicalType(&varcharT)

	chunk := CreateDataChunk([]LogicalType{varcharT})
	defer DestroyDataChunk(&chunk)

	vec := DataChunkGetVector(chunk, 0)
	payload := []byte(`{"proto_type":1,"version":3}`)
	VectorAssignByteElement(vec, 0, payload)
	require.Equal(t, IdxT(0), DataChunkGetSize(chunk))

	DataChunkSetSize(chunk, 1)
	require.Equal(t, IdxT(1), DataChunkGetSize(chunk))
}

func TestVectorAssignByteElement_zeroGoAllocPerAssign(t *testing.T) {
	varcharT := CreateLogicalType(TypeVarchar)
	defer DestroyLogicalType(&varcharT)

	chunk := CreateDataChunk([]LogicalType{varcharT})
	defer DestroyDataChunk(&chunk)

	vec := DataChunkGetVector(chunk, 0)
	b := []byte(`{"key":"value","n":42}`)

	const iterations = 512
	allocs := testing.AllocsPerRun(5, func() {
		for i := 0; i < iterations; i++ {
			VectorAssignByteElement(vec, IdxT(i)%VectorSize(), b)
		}
	})
	if allocs > 0 {
		t.Fatalf("VectorAssignByteElement: want 0 Go allocs per %d assigns, got %v", iterations, allocs)
	}
}

func TestVectorAssignStringElementLen_legacyCopyPath(t *testing.T) {
	varcharT := CreateLogicalType(TypeVarchar)
	defer DestroyLogicalType(&varcharT)

	chunk := CreateDataChunk([]LogicalType{varcharT})
	defer DestroyDataChunk(&chunk)

	vec := DataChunkGetVector(chunk, 0)
	// Legacy API: copies via C.CBytes (alloc may not appear in testing.AllocsPerRun).
	VectorAssignStringElementLen(vec, 0, []byte("legacy-safe-copy"))
}

func BenchmarkVectorAssignByteElement(b *testing.B) {
	varcharT := CreateLogicalType(TypeVarchar)
	defer DestroyLogicalType(&varcharT)

	chunk := CreateDataChunk([]LogicalType{varcharT})
	defer DestroyDataChunk(&chunk)

	vec := DataChunkGetVector(chunk, 0)
	payload := []byte(`{"sid":"abc","method":"INVITE"}`)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		VectorAssignByteElement(vec, IdxT(i)%VectorSize(), payload)
	}
}
