package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGeneratedFiles(t *testing.T) {
	for _, goos := range []string{"darwin", "linux", "windows"} {
		t.Run(goos, func(t *testing.T) {
			requireGolden(t, "prebuilt_"+goos+".golden", platformPrebuilt(goos+"-amd64"))
		})
	}
	requireGolden(t, "static_extension_loader.golden", render(staticExtensionLoaderC, registrations))
	name, data := (&release{modulePrefix: "example.com/native"}).rootPrebuilt("linux-amd64")
	require.Equal(t, "prebuilt_linux_amd64.go", name)
	requireGolden(t, "root_prebuilt.golden", data)
}

func requireGolden(t *testing.T, name string, got []byte) {
	t.Helper()
	want, err := os.ReadFile(filepath.Join("testdata", name))
	require.NoError(t, err)
	require.Equal(t, string(want), string(got))
}

func TestArchiveCache(t *testing.T) {
	payload := []byte("native archive")
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		_, _ = w.Write(payload)
	}))
	defer server.Close()
	r := &release{work: t.TempDir()}
	pin := fmt.Sprintf("%x", sha256.Sum256(payload))
	path, digest := r.archive(server.URL, pin)
	require.Equal(t, pin, digest)
	require.Equal(t, pin+".tar.gz", filepath.Base(path))
	r.archive(server.URL, pin)
	require.Equal(t, 1, requests, "a verified cached pin needs no request")

	require.NoError(t, os.WriteFile(path, []byte("corrupt cache"), 0o644))
	r.archive(server.URL, pin)
	require.Equal(t, 2, requests, "a corrupt cache must be replaced")
	require.Equal(t, pin, sha256File(path))

	for _, next := range []string{"first override", "second override"} {
		payload = []byte(next)
		path, digest = r.archive(server.URL, "")
		require.Equal(t, fmt.Sprintf("%x", sha256.Sum256(payload)), digest)
		require.Equal(t, digest+".tar.gz", filepath.Base(path))
	}
	require.Equal(t, 4, requests, "an override must fetch fresh bytes each time")
}

func TestUseAndFetchShared(t *testing.T) {
	for _, platform := range []string{"darwin-arm64", "linux-amd64", "windows-amd64"} {
		t.Run(platform, func(t *testing.T) {
			r := fixtureRelease(t, platform)
			t.Setenv("PLATFORM", platform)
			r.use()
			require.Equal(t, "header\n", string(readFile("include/duckdb.h")))

			// Go resolves the module through the workspace, including paths
			// containing spaces.
			gowork, err := filepath.Abs("go.work")
			require.NoError(t, err)
			cmd := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", r.module(platform))
			cmd.Env = append(os.Environ(), "GOWORK="+gowork)
			output, err := cmd.CombinedOutput()
			require.NoError(t, err, "%s", output)
			// go list reports the replacement as written, with slashes on Windows.
			require.Equal(t, r.native(platform), filepath.Clean(strings.TrimSpace(string(output))))

			r.stageShared(platform) // CRLF in the shared header is accepted.
			require.FileExists(t, filepath.Join(r.native(platform), "static_extension_loader.c"))
			require.NotContains(t, string(readFile(filepath.Join(r.native(platform), "provenance.json"))), "duckdb_version", "overrides must not claim the pinned build")
		})
	}
}

func fixtureRelease(t *testing.T, platform string) *release {
	t.Helper()
	root := filepath.Join(t.TempDir(), "repo with spaces")
	require.NoError(t, os.MkdirAll(filepath.Join(root, "include"), 0o755))
	t.Chdir(root)
	writeFile("LICENSE", []byte("fixture license\n"))
	writeFile("go.mod", fmt.Appendf(nil, "module example.com/bindings\n\ngo 1.24\n\nrequire example.com/native/%s v0.0.1\n", platform))
	static := map[string]string{
		"duckdb.h": "header\n", "duckdb_static_extension.h": "static header\n", "libduckdb_static.a": "library",
	}
	// These are the upstream filenames, independent of the generator's lists.
	for _, name := range []string{"duckdb_httplib", "duckdb_loadable_extensions", "autocomplete_extension", "core_functions_extension", "icu_extension", "json_extension", "parquet_extension", "tpcds_extension", "tpch_extension"} {
		static["lib"+name+".a"] = "library"
	}
	shared := map[string]string{"duckdb.h": "header\r\n", "libduckdb.dylib": "library", "libduckdb.so": "library", "duckdb.dll": "library", "duckdb.lib": "import library"}
	archives := map[string][]byte{
		"/duckdb-static-libs-static-name.tar.gz": tarball(t, static),
		"/duckdb-shared-libs-shared-name.tar.gz": tarball(t, shared),
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if data, ok := archives[req.URL.Path]; ok {
			_, _ = w.Write(data)
		} else {
			http.NotFound(w, req)
		}
	}))
	t.Cleanup(server.Close)
	return &release{
		manifest:     manifest{BaseURL: server.URL, Platforms: map[string]artifact{platform: {Static: "static-name", Shared: "shared-name"}}},
		modulePrefix: "example.com/native", work: filepath.Join(t.TempDir(), "native work"), goVersion: "1.24",
	}
}

func tarball(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, data := range files {
		require.NoError(t, tw.WriteHeader(&tar.Header{Name: "./" + name, Mode: 0o644, Size: int64(len(data))}))
		_, err := tw.Write([]byte(data))
		require.NoError(t, err)
	}
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())
	return buf.Bytes()
}

func TestRejectedArtifacts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("wrong archive"))
	}))
	defer server.Close()
	r := &release{work: t.TempDir()}
	requireFailure(t, "SHA-256", func() { r.archive(server.URL, strings.Repeat("0", 64)) })
	cached, err := filepath.Glob(filepath.Join(r.work, "downloads", "*"))
	require.NoError(t, err)
	require.Empty(t, cached, "rejected bytes must not enter the cache")

	a, b := filepath.Join(r.work, "a.h"), filepath.Join(r.work, "b.h")
	writeFile(a, []byte("old header\n"))
	writeFile(b, []byte("new header\n"))
	requireFailure(t, "differ", func() { checkSameHeader(a, b) })
}

// requireFailure checks that f aborts the command with an error containing want.
func requireFailure(t *testing.T, want string, f func()) {
	t.Helper()
	defer func() {
		e := recover()
		err, ok := e.(failure)
		require.True(t, ok, "want a failure containing %q, got %v", want, e)
		require.ErrorContains(t, err, want)
	}()
	f()
}
