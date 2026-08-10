//go:build duckdb_nightly

package duckdb_go_bindings

import (
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const minDuckDBSourceIDLength = 7

func openNightlyConnection(t *testing.T, extensionDirectory string) (Connection, func()) {
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

	cleanup := func() {
		Disconnect(&conn)
		Close(&db)
		DestroyConfig(&config)
	}
	return conn, cleanup
}

func TestNightlyArtifactMatchesRequestedDuckDBCommit(t *testing.T) {
	expectedSHA := strings.ToLower(os.Getenv("DUCKDB_SHA"))
	require.Len(t, expectedSHA, 40, "DUCKDB_SHA must be a full commit SHA")
	_, err := hex.DecodeString(expectedSHA)
	require.NoError(t, err, "DUCKDB_SHA must be hexadecimal")

	conn, cleanup := openNightlyConnection(t, "")
	defer cleanup()

	query := fmt.Sprintf(`
		SELECT CASE
			WHEN length(source_id) >= %d
				AND regexp_full_match(source_id, '[0-9a-fA-F]+')
				AND starts_with('%s', lower(source_id))
			THEN 1
			ELSE error(
				'linked DuckDB reports source_id ' || source_id ||
				', expected a hexadecimal prefix of at least %d characters matching %s'
			)
		END
		FROM pragma_version()
	`, minDuckDBSourceIDLength, expectedSHA, minDuckDBSourceIDLength, expectedSHA)

	var result Result
	defer DestroyResult(&result)
	require.Equal(t, StateSuccess, Query(conn, query, &result), ResultError(&result))
	require.Equal(t, int64(1), ValueInt64(&result, 0, 0))
}

func TestNightlyArtifactInstallsAndLoadsHTTPFS(t *testing.T) {
	conn, cleanup := openNightlyConnection(t, t.TempDir())
	defer cleanup()

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
