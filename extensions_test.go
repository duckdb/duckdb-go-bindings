package duckdb_go_bindings

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func getTestDir(t *testing.T) string {
	path, err := os.Getwd()
	require.NoError(t, err)
	return filepath.Join(path, "test") + "/"
}

// TestBuiltinExtensions checks the extensions that every library flavor links:
// registered by the static loader, or by the shared library itself.
func TestBuiltinExtensions(t *testing.T) {
	conn := openTestConnection(t)
	requireBuiltinExtensions(t, conn, "autocomplete", "core_functions", "icu", "json", "parquet")

	// to_minutes comes from core_functions, so this also exercises a function
	// supplied by a built-in extension.
	var res Result
	defer DestroyResult(&res)
	require.Equal(t, StateSuccess, Query(conn, "SELECT to_minutes(5)", &res), ResultError(&res))
}

// requireBuiltinExtensions checks that the named extensions are statically
// linked and loaded as soon as a database is opened, without INSTALL or LOAD.
func requireBuiltinExtensions(t *testing.T, conn Connection, names ...string) {
	t.Helper()

	query := fmt.Sprintf(`
		SELECT error('not statically linked and loaded: ' || string_agg(name, ', ' ORDER BY name))
		FROM unnest(['%s']) AS t(name)
		WHERE name NOT IN (
			SELECT extension_name FROM duckdb_extensions()
			WHERE loaded AND install_mode = 'STATICALLY_LINKED'
		)
		HAVING count(*) > 0
	`, strings.Join(names, "', '"))
	var res Result
	defer DestroyResult(&res)
	require.Equal(t, StateSuccess, Query(conn, query, &res), ResultError(&res))
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
