package duckdb_go_bindings

import (
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func tempExtensionDirectory(t *testing.T) string {
	t.Helper()

	dir, err := os.MkdirTemp("", "duckdb-test-ext-*")
	require.NoError(t, err)
	t.Cleanup(func() {
		// DuckDB keeps loaded extension DLLs open for the process lifetime, so
		// Windows may not release the directory until go test exits.
		if err := os.RemoveAll(dir); err != nil && runtime.GOOS != "windows" {
			t.Error(err)
		}
	})
	return dir
}

func openTestConnection(t *testing.T) Connection {
	t.Helper()
	t.Cleanup(VerifyAllocationCounters)

	var config Config
	require.Equal(t, StateSuccess, CreateConfig(&config))
	t.Cleanup(func() { DestroyConfig(&config) })
	require.Equal(t, StateSuccess, SetConfig(config, "extension_directory", tempExtensionDirectory(t)))

	var db Database
	var errMsg string
	require.Equal(t, StateSuccess, OpenExt(":memory:", &db, config, &errMsg), errMsg)
	t.Cleanup(func() { Close(&db) })

	var conn Connection
	require.Equal(t, StateSuccess, Connect(db, &conn))
	t.Cleanup(func() { Disconnect(&conn) })
	return conn
}
