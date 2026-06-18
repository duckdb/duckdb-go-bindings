//go:build duckdb_arrow

package duckdb_go_bindings

import (
	"testing"

	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/stretchr/testify/require"
)

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
		VALUES (1, 'foo'), (2, 'bar'), (3, null)
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
	defer func() {
		for i := range colCount {
			DestroyLogicalType(&types[i])
		}
	}()
	// get arrow options
	opt := ArrowOptions{}
	ConnectionGetArrowOptions(conn, &opt)
	defer DestroyArrowOptions(&opt)

	// get schema
	schema, ed := NewArrowSchema(opt, types, names)
	defer DestroyErrorData(&ed)
	require.False(t, ErrorDataHasError(ed))
	require.Equal(t, 2, int(schema.NumFields()))

	// get data chunk
	chunk := FetchChunk(res)
	if chunk.Ptr == nil {
		t.Fatal("no data")
	}
	defer DestroyDataChunk(&chunk)

	rec, ed := DataChunkToArrowArray(opt, schema, chunk)
	defer DestroyErrorData(&ed)
	require.False(t, ErrorDataHasError(ed))
	defer rec.Release()

	require.Equal(t, 2, int(rec.NumCols()))
	require.Equal(t, 3, int(rec.NumRows()))
	if v, ok := rec.Column(0).(*array.Int32); ok {
		require.Equal(t, []int32{1, 2, 3}, v.Int32Values())
	} else {
		t.Fatalf("expected int32 column, got %T", rec.Column(0))
	}
	if v, ok := rec.Column(1).(*array.String); ok {
		require.Equal(t, "foo", v.Value(0))
		require.Equal(t, "bar", v.Value(1))
		require.True(t, v.IsNull(2))
	} else {
		t.Fatalf("expected String column, got %T", rec.Column(1))
	}

	newRec := array.NewRecordBatch(schema, rec.Columns(), rec.NumRows())
	defer newRec.Release()

	convSchema, ed := SchemaFromArrow(conn, newRec.Schema())
	defer DestroyErrorData(&ed)
	require.False(t, ErrorDataHasError(ed))
	defer DestroyArrowConvertedSchema(&convSchema)

	dataChunk, ed := DataChunkFromArrow(conn, newRec, convSchema)
	defer DestroyErrorData(&ed)
	require.False(t, ErrorDataHasError(ed))
	defer DestroyDataChunk(&dataChunk)

	cc := DataChunkGetColumnCount(dataChunk)
	require.Equal(t, colCount, cc)

	rc := DataChunkGetSize(dataChunk)
	require.Equal(t, IdxT(rec.NumRows()), rc)
}
