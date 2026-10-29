//go:build duckdb_arrow

package duckdb_go_bindings

import (
	"testing"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/stretchr/testify/require"
)

func requireArrowConvertedSchema(tb testing.TB, conn Connection, schema *arrow.Schema) ArrowConvertedSchema {
	tb.Helper()

	converted, ed := SchemaFromArrow(conn, schema)
	if ErrorDataHasError(ed) {
		msg := ErrorDataMessage(ed)
		DestroyErrorData(&ed)
		DestroyArrowConvertedSchema(&converted)
		tb.Fatalf("SchemaFromArrow failed: %s", msg)
	}
	DestroyErrorData(&ed)
	if converted.Ptr == nil {
		tb.Fatal("SchemaFromArrow returned an empty converted schema")
	}
	return converted
}

func requireDataChunkFromArrow(t *testing.T, conn Connection, rec arrow.RecordBatch, converted ArrowConvertedSchema) DataChunk {
	t.Helper()

	// Keep rec paired with the Arrow schema used to create converted; the native
	// conversion assumes their layouts are compatible.
	chunk, ed := DataChunkFromArrow(conn, rec, converted)
	if ErrorDataHasError(ed) {
		msg := ErrorDataMessage(ed)
		DestroyErrorData(&ed)
		DestroyDataChunk(&chunk)
		t.Fatalf("DataChunkFromArrow failed: %s", msg)
	}
	DestroyErrorData(&ed)
	if chunk.Ptr == nil {
		t.Fatal("DataChunkFromArrow returned an empty data chunk")
	}
	return chunk
}

func requireArrowRoundTrip(
	t *testing.T,
	conn Connection,
	options ArrowOptions,
	converted ArrowConvertedSchema,
	input arrow.RecordBatch,
	expected arrow.RecordBatch,
) {
	t.Helper()

	chunk := requireDataChunkFromArrow(t, conn, input, converted)
	defer DestroyDataChunk(&chunk)
	require.Equal(t, IdxT(input.NumCols()), DataChunkGetColumnCount(chunk))
	require.Equal(t, IdxT(input.NumRows()), DataChunkGetSize(chunk))

	// Import with the expected schema so dictionary inputs can be compared to plain values.
	out, ed := DataChunkToArrowArray(options, expected.Schema(), chunk)
	if ErrorDataHasError(ed) {
		msg := ErrorDataMessage(ed)
		DestroyErrorData(&ed)
		if out != nil {
			out.Release()
		}
		t.Fatalf("DataChunkToArrowArray failed: %s", msg)
	}
	DestroyErrorData(&ed)
	if out == nil {
		t.Fatal("DataChunkToArrowArray returned an empty record batch")
	}
	defer out.Release()

	require.Equal(t, expected.NumCols(), out.NumCols())
	require.Equal(t, expected.NumRows(), out.NumRows())
	for i := 0; i < int(expected.NumCols()); i++ {
		require.True(t, array.Equal(expected.Column(i), out.Column(i)), "column %d did not round-trip", i)
	}
}

func newPrimitiveStringRecord(
	schema *arrow.Schema,
	ints []int32,
	intsValid []bool,
	strings []string,
	stringsValid []bool,
) arrow.RecordBatch {
	builder := array.NewRecordBuilder(memory.DefaultAllocator, schema)
	defer builder.Release()
	builder.Field(0).(*array.Int32Builder).AppendValues(ints, intsValid)
	builder.Field(1).(*array.StringBuilder).AppendValues(strings, stringsValid)
	return builder.NewRecordBatch()
}

func newDictionaryStringRecord(
	t *testing.T,
	schema *arrow.Schema,
	values []string,
	valid []bool,
) arrow.RecordBatch {
	t.Helper()

	builder := array.NewDictionaryBuilder(
		memory.DefaultAllocator,
		schema.Field(0).Type.(*arrow.DictionaryType),
	)
	defer builder.Release()
	strings := builder.(*array.BinaryDictionaryBuilder)
	for i, value := range values {
		if !valid[i] {
			strings.AppendNull()
			continue
		}
		require.NoError(t, strings.AppendString(value))
	}
	dictionary := strings.NewDictionaryArray()
	defer dictionary.Release()
	return array.NewRecordBatch(schema, []arrow.Array{dictionary}, int64(len(values)))
}

func newStringRecord(schema *arrow.Schema, values []string, valid []bool) arrow.RecordBatch {
	builder := array.NewRecordBuilder(memory.DefaultAllocator, schema)
	defer builder.Release()
	builder.Field(0).(*array.StringBuilder).AppendValues(values, valid)
	return builder.NewRecordBatch()
}

func TestDataChunkFromArrowCachedSchemaPrimitiveAndString(t *testing.T) {
	conn := openTestConnection(t)
	schema := arrow.NewSchema([]arrow.Field{
		{Name: "id", Type: arrow.PrimitiveTypes.Int32, Nullable: true},
		{Name: "label", Type: arrow.BinaryTypes.String, Nullable: true},
	}, nil)
	rec1 := newPrimitiveStringRecord(
		schema,
		[]int32{1, 2, 3, 4},
		[]bool{true, false, true, true},
		[]string{"alpha", "", "ignored", "omega"},
		[]bool{true, true, false, true},
	)
	defer rec1.Release()
	rec2 := newPrimitiveStringRecord(
		schema,
		[]int32{5, 6, 7},
		[]bool{true, true, false},
		[]string{"", "beta", ""},
		[]bool{true, true, false},
	)
	defer rec2.Release()

	converted := requireArrowConvertedSchema(t, conn, schema)
	defer DestroyArrowConvertedSchema(&converted)
	var options ArrowOptions
	ConnectionGetArrowOptions(conn, &options)
	defer DestroyArrowOptions(&options)

	requireArrowRoundTrip(t, conn, options, converted, rec1, rec1)
	requireArrowRoundTrip(t, conn, options, converted, rec2, rec2)
}

func TestDataChunkFromArrowCachedSchemaNestedListAndStruct(t *testing.T) {
	conn := openTestConnection(t)
	schema := arrow.NewSchema([]arrow.Field{
		{Name: "items", Type: arrow.ListOf(arrow.PrimitiveTypes.Int32), Nullable: true},
		{Name: "person", Type: arrow.StructOf(
			arrow.Field{Name: "name", Type: arrow.BinaryTypes.String, Nullable: true},
			arrow.Field{Name: "score", Type: arrow.PrimitiveTypes.Int32, Nullable: true},
		), Nullable: true},
	}, nil)

	builder := array.NewRecordBuilder(memory.DefaultAllocator, schema)
	listBuilder := builder.Field(0).(*array.ListBuilder)
	structBuilder := builder.Field(1).(*array.StructBuilder)

	listBuilder.Append(true)
	listBuilder.ValueBuilder().(*array.Int32Builder).AppendValues([]int32{1, 2}, nil)
	structBuilder.Append(true)
	structBuilder.FieldBuilder(0).(*array.StringBuilder).Append("ada")
	structBuilder.FieldBuilder(1).(*array.Int32Builder).Append(42)
	listBuilder.Append(false)
	structBuilder.Append(true)
	structBuilder.FieldBuilder(0).(*array.StringBuilder).Append("")
	structBuilder.FieldBuilder(1).(*array.Int32Builder).AppendNull()
	rec1 := builder.NewRecordBatch()
	defer rec1.Release()

	listBuilder.Append(true)
	structBuilder.Append(true)
	structBuilder.FieldBuilder(0).(*array.StringBuilder).Append("")
	structBuilder.FieldBuilder(1).(*array.Int32Builder).Append(-1)
	listBuilder.Append(true)
	listBuilder.ValueBuilder().(*array.Int32Builder).Append(3)
	structBuilder.Append(true)
	structBuilder.FieldBuilder(0).(*array.StringBuilder).AppendNull()
	structBuilder.FieldBuilder(1).(*array.Int32Builder).Append(7)
	rec2 := builder.NewRecordBatch()
	defer rec2.Release()
	builder.Release()

	converted := requireArrowConvertedSchema(t, conn, schema)
	defer DestroyArrowConvertedSchema(&converted)
	var options ArrowOptions
	ConnectionGetArrowOptions(conn, &options)
	defer DestroyArrowOptions(&options)

	requireArrowRoundTrip(t, conn, options, converted, rec1, rec1)
	requireArrowRoundTrip(t, conn, options, converted, rec2, rec2)
}

func TestDataChunkFromArrowCachedSchemaDictionaryStrings(t *testing.T) {
	conn := openTestConnection(t)
	dictionaryType := &arrow.DictionaryType{
		IndexType: arrow.PrimitiveTypes.Int8,
		ValueType: arrow.BinaryTypes.String,
	}
	schema := arrow.NewSchema([]arrow.Field{
		{Name: "label", Type: dictionaryType, Nullable: true},
	}, nil)
	expectedSchema := arrow.NewSchema([]arrow.Field{
		{Name: "label", Type: arrow.BinaryTypes.String, Nullable: true},
	}, nil)
	values1 := []string{"alpha", "beta", "alpha", ""}
	valid1 := []bool{true, true, true, false}
	rec1 := newDictionaryStringRecord(t, schema, values1, valid1)
	defer rec1.Release()
	expected1 := newStringRecord(expectedSchema, values1, valid1)
	defer expected1.Release()
	values2 := []string{"gamma", "", "gamma"}
	valid2 := []bool{true, true, true}
	rec2 := newDictionaryStringRecord(t, schema, values2, valid2)
	defer rec2.Release()
	expected2 := newStringRecord(expectedSchema, values2, valid2)
	defer expected2.Release()

	converted := requireArrowConvertedSchema(t, conn, schema)
	defer DestroyArrowConvertedSchema(&converted)
	var options ArrowOptions
	ConnectionGetArrowOptions(conn, &options)
	defer DestroyArrowOptions(&options)

	requireArrowRoundTrip(t, conn, options, converted, rec1, expected1)
	requireArrowRoundTrip(t, conn, options, converted, rec2, expected2)
}

func TestDataChunkFromArrowCachedSchemaEmptyThenNonEmpty(t *testing.T) {
	conn := openTestConnection(t)
	schema := arrow.NewSchema([]arrow.Field{
		{Name: "id", Type: arrow.PrimitiveTypes.Int32},
		{Name: "label", Type: arrow.BinaryTypes.String, Nullable: true},
	}, nil)
	empty := newPrimitiveStringRecord(schema, nil, nil, nil, nil)
	defer empty.Release()
	nonEmpty := newPrimitiveStringRecord(
		schema,
		[]int32{1, 2},
		nil,
		[]string{"one", ""},
		[]bool{true, true},
	)
	defer nonEmpty.Release()

	converted := requireArrowConvertedSchema(t, conn, schema)
	defer DestroyArrowConvertedSchema(&converted)
	var options ArrowOptions
	ConnectionGetArrowOptions(conn, &options)
	defer DestroyArrowOptions(&options)

	requireArrowRoundTrip(t, conn, options, converted, empty, empty)
	requireArrowRoundTrip(t, conn, options, converted, nonEmpty, nonEmpty)
}

func BenchmarkDataChunkFromArrowCachedSchema(b *testing.B) {
	metadata := arrow.NewMetadata(
		[]string{"source", "batch_layout"},
		[]string{"benchmark", "four-equal-schema-batches"},
	)
	schema := arrow.NewSchema([]arrow.Field{
		{Name: "id", Type: arrow.PrimitiveTypes.Int32},
		{Name: "label", Type: arrow.BinaryTypes.String, Nullable: true},
	}, &metadata)
	records := make([]arrow.RecordBatch, 4)
	for batch := range records {
		builder := array.NewRecordBuilder(memory.DefaultAllocator, schema)
		ids := builder.Field(0).(*array.Int32Builder)
		labels := builder.Field(1).(*array.StringBuilder)
		for row := 0; row < 256; row++ {
			ids.Append(int32(batch*256 + row))
			switch {
			case row%13 == 0:
				labels.AppendNull()
			case row%17 == 0:
				labels.Append("")
			default:
				labels.Append("cached-schema-row")
			}
		}
		records[batch] = builder.NewRecordBatch()
		builder.Release()
	}
	b.Cleanup(func() {
		for i := range records {
			records[i].Release()
			records[i] = nil
		}
		records = nil
	})

	_, conn := benchMustOpen(b)
	converted := requireArrowConvertedSchema(b, conn, schema)
	b.Cleanup(func() { DestroyArrowConvertedSchema(&converted) })

	const expectedRows = IdxT(4 * 256)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var rows IdxT
		for _, rec := range records {
			chunk, errData := DataChunkFromArrow(conn, rec, converted)
			if ErrorDataHasError(errData) {
				b.StopTimer()
				msg := ErrorDataMessage(errData)
				DestroyErrorData(&errData)
				DestroyDataChunk(&chunk)
				b.Fatalf("DataChunkFromArrow failed: %s", msg)
			}
			DestroyErrorData(&errData)
			if chunk.Ptr == nil {
				b.StopTimer()
				DestroyDataChunk(&chunk)
				b.Fatal("DataChunkFromArrow returned an empty data chunk")
			}
			rows += DataChunkGetSize(chunk)
			DestroyDataChunk(&chunk)
		}
		if rows != expectedRows {
			b.StopTimer()
			b.Fatalf("converted %d rows, want %d", rows, expectedRows)
		}
	}
}

func TestNewArrowSchemaLengths(t *testing.T) {
	conn := openTestConnection(t)
	var options ArrowOptions
	ConnectionGetArrowOptions(conn, &options)
	defer DestroyArrowOptions(&options)
	integerT := CreateLogicalType(TypeInteger)
	defer DestroyLogicalType(&integerT)
	varcharT := CreateLogicalType(TypeVarchar)
	defer DestroyLogicalType(&varcharT)
	types := []LogicalType{integerT, varcharT}

	for _, names := range [][]string{nil, {"a", "b", "c"}} {
		schema, ed := NewArrowSchema(options, types, names)
		defer DestroyErrorData(&ed)
		require.Nil(t, schema, names)
		require.Equal(t, ErrorTypeInvalidInput, ErrorDataErrorType(ed), names)
	}

	schema, ed := NewArrowSchema(options, nil, nil)
	defer DestroyErrorData(&ed)
	require.False(t, ErrorDataHasError(ed), ErrorDataMessage(ed))
	require.Zero(t, schema.NumFields())
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
