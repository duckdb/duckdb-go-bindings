package main

import (
	"cmp"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDropSums(t *testing.T) {
	path := filepath.Join(t.TempDir(), "go.sum")
	writeFile(path, []byte("example.com/native/linux-amd64 v1.0.0 h1:archive\n"+
		"example.com/native-tools v1.0.0 h1:keep\n"+
		"example.com/native/linux-amd64 v1.0.0/go.mod h1:mod\n"+
		"example.com/other v1.0.0 h1:keep"))
	dropSums(path, "example.com/native/")
	require.Equal(t, "example.com/native-tools v1.0.0 h1:keep\nexample.com/other v1.0.0 h1:keep", string(readFile(path)))
}

func TestLoadOverride(t *testing.T) {
	r := fixtureRelease(t, "linux-amd64")
	r.Origin = "https://example.com"
	r.DuckDBVersion, r.Source = "v2.0.0-pin", "pinned-source"
	r.Platforms["linux-amd64"] = artifact{StaticSHA256: "static-pin", SharedSHA256: "shared-pin"}
	writeJSON(manifestFile, r.manifest)
	t.Setenv("LIB_WORK", r.work)
	t.Setenv("PLATFORM", "linux-amd64")
	for _, tc := range []struct {
		name, baseURL, version, source, wantVersion, wantSource string
	}{
		{"pinned defaults", "", "", "", r.DuckDBVersion, r.Source},
		{"override equal to the pin", r.BaseURL, "", "", "", ""},
		{"explicit build", "https://example.com/custom-build", "custom-version", "custom-source", "custom-version", "custom-source"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("BASE_URL", tc.baseURL)
			t.Setenv("DUCKDB_VERSION", tc.version)
			t.Setenv("DUCKDB_SHA", tc.source)
			got := load()
			require.Equal(t, cmp.Or(tc.baseURL, r.BaseURL), got.BaseURL)
			pins := got.Platforms["linux-amd64"]
			if tc.baseURL == "" {
				require.Equal(t, r.Platforms["linux-amd64"], pins)
			} else {
				require.Empty(t, pins.StaticSHA256+pins.SharedSHA256, "an override drops the checksum pins")
			}
			env := got.testEnv()
			require.Equal(t, tc.wantVersion, env["DUCKDB_VERSION"])
			require.Equal(t, tc.wantSource, env["DUCKDB_SHA"])
		})
	}
}
