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

func TestGeometryLogicalType(t *testing.T) {
	defer VerifyAllocationCounters()

	geometryT := CreateLogicalType(TypeGeometry)
	defer DestroyLogicalType(&geometryT)

	require.Equal(t, TypeGeometry, GetTypeId(geometryT))

	integerT := CreateLogicalType(TypeInteger)
	defer DestroyLogicalType(&integerT)

	require.Empty(t, GeometryTypeGetCRS(integerT))
	require.Empty(t, GeometryTypeGetCRS(geometryT))
}

func TestGeometryTypeGetCRS(t *testing.T) {
	defer VerifyAllocationCounters()

	var config Config
	defer DestroyConfig(&config)
	if CreateConfig(&config) == StateError {
		t.Fail()
	}

	var db Database
	defer Close(&db)

	var errMsg string
	if OpenExt(":memory:", &db, config, &errMsg) == StateError {
		require.Empty(t, errMsg)
	}

	var conn Connection
	defer Disconnect(&conn)
	if Connect(db, &conn) == StateError {
		t.Fail()
	}

	var res Result
	defer DestroyResult(&res)
	if Query(conn, `SELECT NULL::GEOMETRY('OGC:CRS84') AS g`, &res) == StateError {
		t.Fail()
	}

	require.Equal(t, IdxT(1), ColumnCount(&res))
	require.Equal(t, TypeGeometry, ColumnType(&res, 0))

	logicalType := ColumnLogicalType(&res, 0)
	defer DestroyLogicalType(&logicalType)

	require.Equal(t, TypeGeometry, GetTypeId(logicalType))
	require.Contains(t, GeometryTypeGetCRS(logicalType), "CRS84")
}
