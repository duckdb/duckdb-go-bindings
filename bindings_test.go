package duckdb_go_bindings

import (
	"testing"

	"github.com/apache/arrow-go/v18/arrow/array"
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

func TestArrow(t *testing.T) {
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
	if Query(conn, `FROM (
		VALUES (1, 'foo'), (2, 'bar'), (3, 'baz')
	) AS t(a, b)
	`, &res) == StateError {
		t.Fail()
	}

	colCount := ColumnCount(&res)
	require.Equal(t, 2, int(colCount))

	names := make([]string, colCount)
	types := make([]LogicalType, colCount)
	for i := range colCount {
		names[i] = ColumnName(&res, i)
		types[i] = ColumnLogicalType(&res, i)
	}
	// get arrow options
	opt := ArrowOptions{}
	ConnectionGetArrowOptions(conn, &opt)

	// get schema
	schema, err := NewArrowSchema(&opt, types, names)
	require.NoError(t, err)
	require.Equal(t, 2, int(schema.NumFields()))

	// get data chunk
	chunk := FetchChunk(res)
	if chunk.Ptr == nil {
		t.Fatal("no data")
	}
	defer DestroyDataChunk(&chunk)

	rec, err := DataChunkToArrowArray(&opt, schema, &chunk)
	require.NoError(t, err)
	defer rec.Release()

	t.Log(rec.NumRows())
	t.Log(rec.NumCols())

	require.Equal(t, 2, int(rec.NumCols()))

	newRec := array.NewRecordBatch(schema, rec.Columns(), rec.NumRows())
	defer newRec.Release()

	convSchema, err := SchemaFromArrow(conn, newRec.Schema())
	require.NoError(t, err)
	defer DestroyArrowConvertedSchema(convSchema)

	dataChunk, err := DataChunkFromArrow(conn, newRec, convSchema)
	require.NoError(t, err)
	defer DestroyDataChunk(dataChunk)

	cc := DataChunkGetColumnCount(*dataChunk)
	require.Equal(t, colCount, cc)

	rc := DataChunkGetSize(*dataChunk)
	require.Equal(t, IdxT(rec.NumRows()), rc)
}
