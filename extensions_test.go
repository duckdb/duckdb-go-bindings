package duckdb_go_bindings

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func getTestDir(t *testing.T) string {
	path, err := os.Getwd()
	require.NoError(t, err)
	return filepath.Join(path, "test") + "/"
}

// TestBuiltinExtensions ensures that the extensions linked into every
// library flavor are loaded without INSTALL or LOAD.
func TestBuiltinExtensions(t *testing.T) {
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

	for _, name := range []string{"core_functions", "icu", "json", "parquet"} {
		query := fmt.Sprintf(`SELECT error('%[1]s is not statically linked and loaded')
			WHERE NOT EXISTS (
				SELECT 1 FROM duckdb_extensions()
				WHERE extension_name = '%[1]s' AND loaded AND install_mode = 'STATICALLY_LINKED'
			)`, name)
		var res Result
		if Query(conn, query, &res) == StateError {
			t.Error(ResultError(&res))
		}
		DestroyResult(&res)
	}

	// Implemented by core_functions.
	var res Result
	defer DestroyResult(&res)
	if Query(conn, `SELECT to_minutes(5)`, &res) == StateError {
		t.Error(ResultError(&res))
	}
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
