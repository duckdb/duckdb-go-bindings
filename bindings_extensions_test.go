package duckdb_go_bindings

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestOpenSQLiteDB ensures that extension auto install + load works,
// as well as some basic C API functions.
func TestOpenSQLiteDB(t *testing.T) {
	defer VerifyAllocationCounters()

	dsn := "test/pets.sqlite"

	var config Config
	defer DestroyConfig(&config)
	if CreateConfig(&config) == StateError {
		t.Fail()
	}

	var db Database
	defer Close(&db)

	var errMsg string
	if OpenExt(dsn, &db, config, &errMsg) == StateError {
		require.Empty(t, errMsg)
	}

	var conn Connection
	defer Disconnect(&conn)
	if Connect(db, &conn) == StateError {
		t.Fail()
	}

	var res Result
	defer DestroyResult(&res)
	if Query(conn, `SELECT COUNT(*) FROM pets`, &res) == StateError {
		t.Fail()
	}

	colCount := int(ColumnCount(&res))
	require.Equal(t, 1, colCount)

	colType := ColumnType(&res, 0)
	require.Equal(t, TypeBigInt, colType)
}
