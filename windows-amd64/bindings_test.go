package duckdb_go_bindings

import (
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpen(t *testing.T) {
	require.Equal(t, IdxT(2048), VectorSize())
}

func TestOpenDB(t *testing.T) {
	var db Database

	dsn := "sqlite:pets.sqlite"
	parsedDSN, err := url.Parse(dsn)
	require.NoError(t, err)

	config, err := prepareConfig(parsedDSN)
	require.NoError(t, err)
	defer DestroyConfig(&config)

	connStr := getConnString(dsn)
	var errMsg string
	if OpenExt(connStr, &db, config, &errMsg) == StateError {
		Close(&db)
		panic("error")
	}
	fmt.Println("anything went wrong?")
	fmt.Println(errMsg)
}

func getConnString(dsn string) string {
	idx := strings.Index(dsn, "?")
	if idx < 0 {
		idx = len(dsn)
	}
	return dsn[0:idx]
}

func prepareConfig(parsedDSN *url.URL) (Config, error) {
	var config Config
	if CreateConfig(&config) == StateError {
		DestroyConfig(&config)
		panic("error")
	}

	if err := setConfigOption(config, "duckdb_api", "go"); err != nil {
		return config, err
	}

	// Early-out, if the DSN does not contain configuration options.
	if len(parsedDSN.RawQuery) == 0 {
		return config, nil
	}

	for k, v := range parsedDSN.Query() {
		if len(v) == 0 {
			continue
		}
		if err := setConfigOption(config, k, v[0]); err != nil {
			return config, err
		}
	}

	return config, nil
}

func setConfigOption(config Config, name string, option string) error {
	if SetConfig(config, name, option) == StateError {
		DestroyConfig(&config)
		panic("error")
	}
	return nil
}
