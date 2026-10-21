//go:build duckdb_nightly

package duckdb_go_bindings

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNightlyArtifactInstallsAndLoadsHTTPFS(t *testing.T) {
	conn := openTestConnection(t)

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
