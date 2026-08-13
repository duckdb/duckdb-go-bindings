//go:build duckdb_nightly

package duckdb_go_bindings

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const minDuckDBSourceIDLength = 7

func openNightlyConnection(t *testing.T, extensionDirectory string) Connection {
	t.Helper()

	var config Config
	require.Equal(t, StateSuccess, CreateConfig(&config))
	if extensionDirectory != "" {
		require.Equal(t, StateSuccess, SetConfig(config, "extension_directory", extensionDirectory))
	}

	var db Database
	var errMsg string
	require.Equal(t, StateSuccess, OpenExt(":memory:", &db, config, &errMsg), errMsg)

	var conn Connection
	require.Equal(t, StateSuccess, Connect(db, &conn))

	t.Cleanup(func() {
		Disconnect(&conn)
		Close(&db)
		DestroyConfig(&config)
	})
	return conn
}

func TestNightlyArtifactMatchesRequestedDuckDBCommit(t *testing.T) {
	expectedSHA := strings.ToLower(os.Getenv("DUCKDB_SHA"))
	require.Regexp(t, `^[0-9a-f]{40}$`, expectedSHA, "DUCKDB_SHA must be a full hexadecimal commit SHA")

	conn := openNightlyConnection(t, "")

	// source_id is a commit prefix, so expectedSHA (validated hexadecimal above)
	// must start with it. error() reports the observed source_id on a mismatch.
	query := fmt.Sprintf(`
		SELECT CASE
			WHEN length(source_id) >= %[1]d AND starts_with('%[2]s', lower(source_id))
			THEN 1
			ELSE error(
				'linked DuckDB reports source_id ' || source_id ||
				', expected a hexadecimal prefix of at least %[1]d characters matching %[2]s'
			)
		END
		FROM pragma_version()
	`, minDuckDBSourceIDLength, expectedSHA)

	var result Result
	defer DestroyResult(&result)
	require.Equal(t, StateSuccess, Query(conn, query, &result), ResultError(&result))
	require.Equal(t, int64(1), ValueInt64(&result, 0, 0))
}

func TestNightlyArtifactInstallsAndLoadsHTTPFS(t *testing.T) {
	conn := openNightlyConnection(t, t.TempDir())

	var result Result
	defer DestroyResult(&result)
	state := Query(conn, `
		FORCE INSTALL httpfs;
		LOAD httpfs;
		SELECT CAST(installed AND loaded AS BIGINT)
		FROM duckdb_extensions()
		WHERE extension_name = 'httpfs';
	`, &result)
	require.Equal(t, StateSuccess, state, ResultError(&result))
	require.Equal(t, int64(1), ValueInt64(&result, 0, 0))
}
