package duckdb_go_bindings

import (
	"fmt"
	"os"
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

func TestNamedLogicalTypeLengths(t *testing.T) {
	defer VerifyAllocationCounters()
	integerT := CreateLogicalType(TypeInteger)
	defer DestroyLogicalType(&integerT)
	varcharT := CreateLogicalType(TypeVarchar)
	defer DestroyLogicalType(&varcharT)
	types := []LogicalType{integerT, varcharT}

	for _, names := range [][]string{{"a"}, {"a", "b", "c"}} {
		require.Nil(t, CreateStructType(types, names).Ptr, names)
		require.Nil(t, CreateUnionType(types, names).Ptr, names)
	}

	structT := CreateStructType(types, []string{"a", "b"})
	defer DestroyLogicalType(&structT)
	require.NotNil(t, structT.Ptr)
	unionT := CreateUnionType(types, []string{"a", "b"})
	defer DestroyLogicalType(&unionT)
	require.NotNil(t, unionT.Ptr)
}

func TestAppenderCreateQueryColumnNames(t *testing.T) {
	conn := openTestConnection(t)
	integerT := CreateLogicalType(TypeInteger)
	defer DestroyLogicalType(&integerT)
	varcharT := CreateLogicalType(TypeVarchar)
	defer DestroyLogicalType(&varcharT)
	types := []LogicalType{integerT, varcharT}

	var result Result
	defer DestroyResult(&result)
	require.Equal(t, StateSuccess, Query(conn, "CREATE TABLE target (i INTEGER, s VARCHAR)", &result))

	for _, names := range [][]string{{"a"}, {"a", "b", "c"}} {
		var appender Appender
		require.Equal(t, StateError, AppenderCreateQuery(conn, "INSERT INTO target SELECT * FROM appended_data", types, "", names, &appender), names)
		require.Nil(t, appender.Ptr, names)
	}

	var input Result
	defer DestroyResult(&input)
	require.Equal(t, StateSuccess, Query(conn, "SELECT 42::INTEGER, 'hello'::VARCHAR", &input))
	chunk := FetchChunk(input)
	defer DestroyDataChunk(&chunk)

	// Without column names, DuckDB names the columns col1, col2, ...
	for _, tc := range []struct {
		names   []string
		columns string
	}{
		{nil, "col1, col2"},
		{[]string{"a", "b"}, "a, b"},
	} {
		var appender Appender
		defer AppenderDestroy(&appender)
		state := AppenderCreateQuery(conn, "INSERT INTO target SELECT "+tc.columns+" FROM appended_data", types, "", tc.names, &appender)
		require.Equal(t, StateSuccess, state, AppenderError(appender))
		require.Equal(t, IdxT(2), AppenderColumnCount(appender))
		require.Equal(t, StateSuccess, AppendDataChunk(appender, chunk))
		require.Equal(t, StateSuccess, AppenderFlush(appender))
	}

	var rows Result
	defer DestroyResult(&rows)
	require.Equal(t, StateSuccess, Query(conn, "SELECT count(*) FROM target WHERE i = 42 AND s = 'hello'", &rows))
	require.Equal(t, int64(2), ValueInt64(&rows, 0, 0))
}

func TestLibraryVersion(t *testing.T) {
	defer VerifyAllocationCounters()
	v := LibraryVersion()
	require.NotEmpty(t, v)
	if expected := os.Getenv("DUCKDB_VERSION"); expected != "" {
		require.Equal(t, expected, v)
	}
}

func TestLinkedSourceID(t *testing.T) {
	expectedSHA := os.Getenv("DUCKDB_SHA")
	if expectedSHA == "" {
		t.Skip("Set DUCKDB_SHA to check the linked source ID")
	}
	require.Regexp(t, `^[0-9a-f]{40}$`, expectedSHA, "DUCKDB_SHA must be a full lowercase hexadecimal commit SHA")

	conn := openTestConnection(t)

	// source_id is a commit prefix. Report it if it doesn't match the requested build.
	query := fmt.Sprintf(`
		SELECT CASE
			WHEN length(source_id) >= 7 AND starts_with('%[1]s', lower(source_id))
			THEN 1
			ELSE error('linked DuckDB reports source_id ' || source_id || ', expected a prefix of %[1]s')
		END
		FROM pragma_version()
	`, expectedSHA)
	var result Result
	defer DestroyResult(&result)
	require.Equal(t, StateSuccess, Query(conn, query, &result), ResultError(&result))
	require.Equal(t, int64(1), ValueInt64(&result, 0, 0))
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

func TestVariantLogicalType(t *testing.T) {
	defer VerifyAllocationCounters()

	variantT := CreateLogicalType(TypeVariant)
	defer DestroyLogicalType(&variantT)

	require.Equal(t, TypeVariant, GetTypeId(variantT))
}
