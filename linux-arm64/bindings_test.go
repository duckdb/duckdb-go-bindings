package duckdb_go_bindings

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestVectorSize ensures that linking works.
func TestVectorSize(t *testing.T) {
	defer VerifyAllocationCounters()
	require.Equal(t, IdxT(2048), VectorSize())
}

// TestCreateDataChunk ensures that we allocate C arrays correctly.
func TestCreateDataChunk(t *testing.T) {
	defer VerifyAllocationCounters()

	tinyIntT := CreateLogicalType(TypeTinyInt)
	defer DestroyLogicalType(&tinyIntT)

	varcharT := CreateLogicalType(TypeVarchar)
	defer DestroyLogicalType(&varcharT)

	var types []LogicalType
	types = append(types, tinyIntT, varcharT)

	structT := CreateStructType(types, []string{"c1", "c2"})
	defer DestroyLogicalType(&structT)

	types = append(types, structT)
	chunk := CreateDataChunk(types)
	defer DestroyDataChunk(&chunk)
}

func TestLibraryVersion(t *testing.T) {
	defer VerifyAllocationCounters()
	v := LibraryVersion()
	require.NotEmpty(t, v)
}
