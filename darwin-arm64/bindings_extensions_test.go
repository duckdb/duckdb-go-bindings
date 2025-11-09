package duckdb_go_bindings

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func getTestDir(t *testing.T) string {
	path, err := os.Getwd()
	require.NoError(t, err)
	lastIndex := strings.LastIndex(path, "duckdb-go-bindings")
	return path[:lastIndex] + "duckdb-go-bindings/test/"
}

// TestOpenSQLiteDB ensures that extension auto install + load works,
// as well as some basic C API functions.
func TestOpenSQLiteDB(t *testing.T) {
	defer VerifyAllocationCounters()

	dsn := getTestDir(t) + "pets.sqlite"

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
