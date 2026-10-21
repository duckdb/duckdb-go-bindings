package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"text/template"
	"time"
)

// failure aborts the command; main reports it. No step of a release can be
// recovered from, so helpers fail directly instead of returning errors.
type failure struct{ error }

func check(err error) {
	if err != nil {
		panic(failure{err})
	}
}

func fail(format string, args ...any) { check(fmt.Errorf(format, args...)) }

func mkdir(dirs ...string) {
	for _, dir := range dirs {
		check(os.MkdirAll(dir, 0o755))
	}
}

func readFile(path string) []byte {
	data, err := os.ReadFile(path)
	check(err)
	return data
}

func writeFile(path string, data []byte) { check(os.WriteFile(path, data, 0o644)) }

func create(path string) *os.File {
	f, err := os.Create(path)
	check(err)
	return f
}

func writeFrom(path string, r io.Reader) {
	f := create(path)
	_, err := io.Copy(f, r)
	check(err)
	check(f.Close())
}

func copyFile(src, dst string) {
	f, err := os.Open(src)
	check(err)
	defer f.Close()
	writeFrom(dst, f)
}

// writeJSON writes v as indented JSON with sorted map keys, like jq -S.
func writeJSON(path string, v any) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	check(enc.Encode(v))
	writeFile(path, buf.Bytes())
}

func render(t *template.Template, data any) []byte {
	var buf bytes.Buffer
	check(t.Execute(&buf, data))
	return buf.Bytes()
}

// sha256File returns the hex SHA-256 of a file, or "" if it doesn't exist.
func sha256File(path string) string {
	f, err := os.Open(path)
	if errors.Is(err, fs.ErrNotExist) {
		return ""
	}
	check(err)
	defer f.Close()
	h := sha256.New()
	_, err = io.Copy(h, f)
	check(err)
	return hex.EncodeToString(h.Sum(nil))
}

// checkSameHeader fails unless two copies of duckdb.h match apart from line
// endings.
func checkSameHeader(a, b string) {
	cr := []byte("\r")
	if !bytes.Equal(bytes.ReplaceAll(readFile(a), cr, nil), bytes.ReplaceAll(readFile(b), cr, nil)) {
		fail("%s and %s differ", a, b)
	}
}

// dropSums removes the go.sum lines of modules under prefix.
func dropSums(path, prefix string) {
	var kept []byte
	for line := range bytes.Lines(readFile(path)) {
		if !bytes.HasPrefix(line, []byte(prefix)) {
			kept = append(kept, line...)
		}
	}
	writeFile(path, kept)
}

// run runs a command with extra environment variables, streaming its output.
// go commands resolve modules from go.mod alone, never from a local go.work.
func run(env []string, name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Env = append(append(os.Environ(), "GOWORK=off"), env...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		fail("%s %s: %v", name, strings.Join(args, " "), err)
	}
}

// get issues a GET with curl --retry 3 semantics: transport errors and 408,
// 429 and 5xx responses are retried with backoff. The caller closes the body.
func get(url string) *http.Response {
	var err error
	for attempt := range 4 {
		if attempt > 0 {
			time.Sleep(time.Second << (attempt - 1))
		}
		var resp *http.Response
		if resp, err = http.Get(url); err != nil {
			continue
		}
		code := resp.StatusCode
		if code < 500 && code != http.StatusRequestTimeout && code != http.StatusTooManyRequests {
			return resp
		}
		resp.Body.Close()
		err = fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	check(err)
	return nil
}

func httpStatus(url string) int {
	resp := get(url)
	resp.Body.Close()
	return resp.StatusCode
}

func download(url, path string) {
	resp := get(url)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fail("GET %s: %s", url, resp.Status)
	}
	writeFrom(path, resp.Body)
}
