// Command brood packages DuckDB's static libraries as Go modules, one per
// platform, and publishes them with rclone. Run it from the repository root:
//
//	go run ./internal/cmd/brood fetch|fetch-shared|use|env|package|pin|upload
//
// See "Updating native libraries" in the README for the release process.
package main

import (
	"cmp"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
	"golang.org/x/mod/zip"
)

const (
	manifestFile = "duckdb-release.json"
	modulePath   = "libduckdb" // below the origin
)

type manifest struct {
	Version       string              `json:"version"`
	DuckDBVersion string              `json:"duckdb_version"`
	Source        string              `json:"source"`
	Time          string              `json:"time"`
	BaseURL       string              `json:"base_url"`
	Origin        string              `json:"origin"`
	Platforms     map[string]artifact `json:"platforms"`
}

type artifact struct {
	Static       string `json:"static"`
	Shared       string `json:"shared"`
	StaticSHA256 string `json:"static_sha256"`
	SharedSHA256 string `json:"shared_sha256"`
}

// release is the manifest plus the settings derived from it and the environment.
type release struct {
	manifest
	platforms    []string // sorted platform names, e.g. darwin-arm64
	modulePrefix string   // platform module paths without the platform
	work         string   // LIB_WORK, outside the checkout
	goVersion    string   // go directive of the root module
}

func main() {
	commands := map[string]func(*release){
		"fetch":        func(r *release) { r.stage(r.platform()) },
		"fetch-shared": func(r *release) { r.stageShared(r.platform()) },
		"use":          (*release).use,
		"env":          (*release).env,
		"package":      (*release).packageModules,
		"pin":          (*release).pin,
		"upload":       (*release).upload,
	}
	if len(os.Args) != 2 || commands[os.Args[1]] == nil {
		fmt.Fprintln(os.Stderr, "usage: go run ./internal/cmd/brood fetch|fetch-shared|use|env|package|pin|upload (see README)")
		os.Exit(2)
	}
	defer func() {
		if e := recover(); e != nil {
			f, ok := e.(failure)
			if !ok {
				panic(e)
			}
			fmt.Fprintln(os.Stderr, "brood:", f.error)
			os.Exit(1)
		}
	}()
	commands[os.Args[1]](load())
}

func load() *release {
	data, err := os.ReadFile(manifestFile)
	if err != nil {
		fail("%v (run from the repository root)", err)
	}
	r := &release{}
	if err := json.Unmarshal(data, &r.manifest); err != nil {
		fail("%s: %v", manifestFile, err)
	}
	// BASE_URL selects another build, so the pins of the manifest's build don't
	// apply to it.
	if override := os.Getenv("BASE_URL"); override != "" {
		r.BaseURL, r.DuckDBVersion, r.Source = override, "", ""
		for platform, a := range r.Platforms {
			a.StaticSHA256, a.SharedSHA256 = "", ""
			r.Platforms[platform] = a
		}
	}
	r.platforms = slices.Sorted(maps.Keys(r.Platforms))
	r.modulePrefix = strings.TrimPrefix(r.Origin, "https://") + "/" + modulePath
	r.work = workDir()

	gomod, err := modfile.ParseLax("go.mod", readFile("go.mod"), nil)
	check(err)
	r.goVersion = gomod.Go.Version
	return r
}

// workDir creates and resolves LIB_WORK. It must be outside the checkout, so
// downloads and packaged modules never end up in it.
func workDir() string {
	work, err := filepath.Abs(cmp.Or(os.Getenv("LIB_WORK"), filepath.Join(os.TempDir(), "duckdb-go-libraries")))
	check(err)
	mkdir(work)
	work, err = filepath.EvalSymlinks(work)
	check(err)
	root, err := os.Getwd()
	check(err)
	root, err = filepath.EvalSymlinks(root)
	check(err)
	if rel, err := filepath.Rel(root, work); err == nil && filepath.IsLocal(rel) {
		fail("LIB_WORK must be outside the checkout")
	}
	return work
}

func (r *release) module(platform string) string  { return r.modulePrefix + "/" + platform }
func (r *release) native(platform string) string  { return filepath.Join(r.work, "native", platform) }
func (r *release) dynamic(platform string) string { return filepath.Join(r.work, "dynamic", platform) }
func (r *release) header(platform string) string {
	return filepath.Join(r.work, "headers", platform, "duckdb.h")
}
func (r *release) modules() string  { return filepath.Join(r.work, "modules") }
func (r *release) modcache() string { return filepath.Join(r.work, "modcache") }

// localProxy serves the packaged modules, and the other modules from the
// default proxy.
func (r *release) localProxy() string {
	return "file://" + filepath.ToSlash(r.modules()) + ",https://proxy.golang.org"
}

func (r *release) infoURL(platform string) string {
	return r.Origin + "/modules/" + r.module(platform) + "/@v/" + r.Version + ".info"
}

func (r *release) platform() string {
	platform := cmp.Or(os.Getenv("PLATFORM"), runtime.GOOS+"-"+runtime.GOARCH)
	if _, ok := r.Platforms[platform]; !ok {
		fail("%s has no platform %q", manifestFile, platform)
	}
	return platform
}

// env prints the test runner's paths and defaults. Values are quoted for the
// POSIX shell, also on MSYS.
func (r *release) env() {
	values := r.testEnv()
	for _, key := range slices.Sorted(maps.Keys(values)) {
		fmt.Printf("export %s='%s'\n", key, strings.ReplaceAll(values[key], "'", "'\"'\"'"))
	}
}

// testEnv derives the test runner's settings from the same configuration as
// the artifact commands.
func (r *release) testEnv() map[string]string {
	platform := r.platform()
	return map[string]string{
		"PLATFORM":       platform,
		"NATIVE_DIR":     filepath.ToSlash(r.native(platform)),
		"DYNAMIC_DIR":    filepath.ToSlash(r.dynamic(platform)),
		"MODULE_PROXY":   r.localProxy(),
		"MODULE_CACHE":   filepath.ToSlash(r.modcache()),
		"GOPROXY":        cmp.Or(os.Getenv("GOPROXY"), r.Origin+"/modules,https://proxy.golang.org"),
		"GONOSUMDB":      cmp.Or(os.Getenv("GONOSUMDB"), r.modulePrefix),
		"DUCKDB_VERSION": cmp.Or(os.Getenv("DUCKDB_VERSION"), r.DuckDBVersion),
		"DUCKDB_SHA":     cmp.Or(os.Getenv("DUCKDB_SHA"), r.Source),
	}
}

// use compiles against a staged build's header and replaces this platform's
// native dependency in a local workspace. Go formats native Windows paths too.
func (r *release) use() {
	platform := r.platform()
	r.stage(platform)
	copyFile(r.header(platform), "include/duckdb.h")
	work := &modfile.WorkFile{Syntax: &modfile.FileSyntax{Name: "go.work"}}
	check(work.AddGoStmt(r.goVersion))
	check(work.AddUse(".", ""))
	check(work.AddReplace(r.module(platform), "", filepath.ToSlash(r.native(platform)), ""))
	writeFile("go.work", modfile.Format(work.Syntax))
}

// packageModules writes every platform module to a module proxy tree in
// LIB_WORK/modules.
func (r *release) packageModules() {
	for _, platform := range r.platforms {
		if r.Platforms[platform].StaticSHA256 == "" {
			fail("%s: packaging requires a pinned checksum; unset BASE_URL", platform)
		}
		// Never replace source pins with repackaged bytes of a published version.
		switch status := httpStatus(r.infoURL(platform)); status {
		case http.StatusNotFound:
		case http.StatusOK:
			fail("%s@%s already exists; choose a new native module version", r.module(platform), r.Version)
		default:
			fail("cannot check %s@%s: HTTP %d", r.module(platform), r.Version, status)
		}
	}

	check(os.RemoveAll(r.modules()))
	for _, platform := range r.platforms {
		r.stage(platform)
		checkSameHeader(r.header(r.platforms[0]), r.header(platform))

		mod := module.Version{Path: r.module(platform), Version: r.Version}
		dir := filepath.Join(r.modules(), mod.Path, "@v")
		mkdir(dir)
		zipFile := create(filepath.Join(dir, mod.Version+".zip"))
		check(zip.CreateFromDir(zipFile, mod, r.native(platform)))
		check(zipFile.Close())
		copyFile(filepath.Join(r.native(platform), "go.mod"), filepath.Join(dir, mod.Version+".mod"))
		writeJSON(filepath.Join(dir, mod.Version+".info"), struct{ Version, Time string }{r.Version, r.Time})
	}
}

// pin points the root module at the packaged modules: header, platform
// imports, go.mod and go.sum.
func (r *release) pin() {
	copyFile(r.header(r.platforms[0]), "include/duckdb.h")
	edit := []string{"mod", "edit"}
	for _, platform := range r.platforms {
		writeFile(r.rootPrebuilt(platform))
		edit = append(edit, "-require="+r.module(platform)+"@"+r.Version)
	}
	run(nil, "go", edit...)

	env := []string{"GOMODCACHE=" + r.modcache(), "GOPROXY=" + r.localProxy(), "GONOSUMDB=" + r.modulePrefix}
	// Retries must read this attempt's ZIPs, even if the version hasn't changed.
	run(env, "go", "clean", "-modcache")
	dropSums("go.sum", r.modulePrefix+"/")
	run(env, "go", "mod", "tidy")
}

// upload copies LIB_WORK/modules to LIB_DESTINATION, create-only, then adds the
// go-import pages that let the go command discover the modules.
func (r *release) upload() {
	dest := os.Getenv("LIB_DESTINATION")
	if dest == "" {
		fail("set LIB_DESTINATION to an rclone remote:bucket/prefix")
	}
	rclone := cmp.Or(os.Getenv("RCLONE"), "rclone")
	run(nil, rclone, "copy", r.modules(), path.Join(dest, "modules"),
		"--checksum", "--immutable", "--check-first",
		"--header-upload", "If-None-Match: *",
		"--header-upload", "Cache-Control: public, max-age=31536000, immutable",
		"--s3-upload-cutoff", "1G")
	for _, platform := range r.platforms {
		if status := httpStatus(r.infoURL(platform)); status != http.StatusOK {
			fail("%s: HTTP %d after upload", r.infoURL(platform), status)
		}
	}

	// Each page names its own module path, so the go command trusts it without
	// fetching a page for a shorter prefix.
	for _, platform := range r.platforms {
		page := filepath.Join(r.work, "go-import-"+platform+".html")
		writeFile(page, fmt.Appendf(nil, "<!doctype html><meta name=\"go-import\" content=\"%s mod %s/modules\">\n", r.module(platform), r.Origin))
		run(nil, rclone, "copyto", page, path.Join(dest, modulePath, platform),
			"--header-upload", "Content-Type: text/html; charset=utf-8",
			"--header-upload", "Cache-Control: no-cache")
	}
	fmt.Printf("Published %s: %s/modules\n", r.Version, r.Origin)
}
