// Package main is the bench workload that drives duckdb-go-bindings.
//
// The exact same file is compiled twice (via go.mod replace) — once against the
// upstream module and once against the local fork — and its JSON output on
// stdout is consumed by the compare orchestrator.
//
// It does NOT measure with go's testing/benchmark framework on purpose:
// we want OS-level rusage (user/sys CPU, max RSS) and runtime counters
// (NumCgoCall, MemStats) across the WHOLE process, isolated from go bench
// machinery, with a steady-state warmup phase.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"runtime"
	"sort"
	"syscall"
	"time"

	duckdb "github.com/duckdb/duckdb-go-bindings"
)

type sampleSnapshot struct {
	wallNs   int64
	userUs   int64
	sysUs    int64
	cgoCalls int64
	mallocs  uint64
	bytesAllocCumulative uint64
}

type scenarioSummary struct {
	Name        string  `json:"name"`
	Iters       int     `json:"iters_per_run"`
	Runs        int     `json:"runs"`
	WallNsMed   float64 `json:"wall_ns_median"`
	WallNsMean  float64 `json:"wall_ns_mean"`
	WallNsStd   float64 `json:"wall_ns_stddev"`
	UserUsMed   float64 `json:"user_us_median"`
	SysUsMed    float64 `json:"sys_us_median"`
	CPUUsMed    float64 `json:"cpu_us_median"`
	CGOPerOp    float64 `json:"cgo_calls_per_op_median"`
	GoAllocsOp  float64 `json:"go_mallocs_per_op_median"`
	GoBytesOp   float64 `json:"go_bytes_per_op_median"`
	NsPerOp     float64 `json:"ns_per_op_median"`
	CPUNsPerOp  float64 `json:"cpu_ns_per_op_median"`
}

type benchReport struct {
	Variant     string            `json:"variant"`
	Iterations  int               `json:"iterations"`
	Runs        int               `json:"runs"`
	Warmup      int               `json:"warmup_runs"`
	GoVersion   string            `json:"go_version"`
	GoMaxProcs  int               `json:"go_maxprocs"`
	Scenarios   []scenarioSummary `json:"scenarios"`
}

func snapshot() sampleSnapshot {
	var ru syscall.Rusage
	_ = syscall.Getrusage(syscall.RUSAGE_SELF, &ru)
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	return sampleSnapshot{
		wallNs:   time.Now().UnixNano(),
		userUs:   ru.Utime.Sec*1_000_000 + int64(ru.Utime.Usec),
		sysUs:    ru.Stime.Sec*1_000_000 + int64(ru.Stime.Usec),
		cgoCalls: runtime.NumCgoCall(),
		mallocs:  ms.Mallocs,
		bytesAllocCumulative: ms.TotalAlloc,
	}
}

type scenarioRun struct {
	wallNs   int64
	userUs   int64
	sysUs    int64
	cgoCalls int64
	mallocs  int64
	bytes    int64
}

func runScenario(name string, iters, runs, warmup int, fn func(int)) scenarioSummary {
	for i := 0; i < warmup; i++ {
		fn(iters)
	}
	runtime.GC()
	runs2 := make([]scenarioRun, runs)
	for i := 0; i < runs; i++ {
		runtime.GC()
		before := snapshot()
		fn(iters)
		after := snapshot()
		runs2[i] = scenarioRun{
			wallNs:   after.wallNs - before.wallNs,
			userUs:   after.userUs - before.userUs,
			sysUs:    after.sysUs - before.sysUs,
			cgoCalls: after.cgoCalls - before.cgoCalls,
			mallocs:  int64(after.mallocs - before.mallocs),
			bytes:    int64(after.bytesAllocCumulative - before.bytesAllocCumulative),
		}
	}

	wall := make([]float64, runs)
	user := make([]float64, runs)
	sys := make([]float64, runs)
	cgo := make([]float64, runs)
	mal := make([]float64, runs)
	byt := make([]float64, runs)
	for i, r := range runs2 {
		wall[i] = float64(r.wallNs)
		user[i] = float64(r.userUs)
		sys[i] = float64(r.sysUs)
		cgo[i] = float64(r.cgoCalls) / float64(iters)
		mal[i] = float64(r.mallocs) / float64(iters)
		byt[i] = float64(r.bytes) / float64(iters)
	}

	itersF := float64(iters)
	cpuUs := make([]float64, runs)
	for i := range runs2 {
		cpuUs[i] = user[i] + sys[i]
	}

	return scenarioSummary{
		Name:       name,
		Iters:      iters,
		Runs:       runs,
		WallNsMed:  median(wall),
		WallNsMean: mean(wall),
		WallNsStd:  stddev(wall),
		UserUsMed:  median(user),
		SysUsMed:   median(sys),
		CPUUsMed:   median(cpuUs),
		CGOPerOp:   median(cgo),
		GoAllocsOp: median(mal),
		GoBytesOp:  median(byt),
		NsPerOp:    median(wall) / itersF,
		CPUNsPerOp: median(cpuUs) * 1000.0 / itersF,
	}
}

func median(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	cp := append([]float64(nil), xs...)
	sort.Float64s(cp)
	n := len(cp)
	if n%2 == 1 {
		return cp[n/2]
	}
	return (cp[n/2-1] + cp[n/2]) / 2
}

func mean(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	var s float64
	for _, x := range xs {
		s += x
	}
	return s / float64(len(xs))
}

func stddev(xs []float64) float64 {
	if len(xs) < 2 {
		return 0
	}
	m := mean(xs)
	var s float64
	for _, x := range xs {
		d := x - m
		s += d * d
	}
	return math.Sqrt(s / float64(len(xs)-1))
}

func openMemoryDB() (duckdb.Database, duckdb.Connection) {
	var db duckdb.Database
	if duckdb.Open(":memory:", &db) != duckdb.StateSuccess {
		fmt.Fprintln(os.Stderr, "duckdb_open :memory: failed")
		os.Exit(2)
	}
	var conn duckdb.Connection
	if duckdb.Connect(db, &conn) != duckdb.StateSuccess {
		fmt.Fprintln(os.Stderr, "duckdb_connect failed")
		os.Exit(2)
	}
	return db, conn
}

func scnOpenClose(iters int) {
	for i := 0; i < iters; i++ {
		var db duckdb.Database
		if duckdb.Open(":memory:", &db) != duckdb.StateSuccess {
			panic("open")
		}
		var conn duckdb.Connection
		if duckdb.Connect(db, &conn) != duckdb.StateSuccess {
			panic("connect")
		}
		duckdb.Disconnect(&conn)
		duckdb.Close(&db)
	}
}

func scnPrepareShort(iters int) {
	db, conn := openMemoryDB()
	defer duckdb.Close(&db)
	defer duckdb.Disconnect(&conn)
	const sql = "SELECT $1::VARCHAR"
	for i := 0; i < iters; i++ {
		var stmt duckdb.PreparedStatement
		if duckdb.Prepare(conn, sql, &stmt) != duckdb.StateSuccess {
			panic("prepare")
		}
		duckdb.DestroyPrepare(&stmt)
	}
}

func scnQuerySelect1(iters int) {
	db, conn := openMemoryDB()
	defer duckdb.Close(&db)
	defer duckdb.Disconnect(&conn)
	const sql = "SELECT 1 AS x"
	for i := 0; i < iters; i++ {
		var res duckdb.Result
		if duckdb.Query(conn, sql, &res) != duckdb.StateSuccess {
			panic("query")
		}
		duckdb.DestroyResult(&res)
	}
}

func scnBindVarchar(iters int) {
	db, conn := openMemoryDB()
	defer duckdb.Close(&db)
	defer duckdb.Disconnect(&conn)
	var stmt duckdb.PreparedStatement
	if duckdb.Prepare(conn, "SELECT $1::VARCHAR WHERE $1 IS NOT NULL", &stmt) != duckdb.StateSuccess {
		panic("prepare bind_varchar")
	}
	defer duckdb.DestroyPrepare(&stmt)
	s := "ingest-tag-value-pair-short"
	for i := 0; i < iters; i++ {
		if duckdb.BindVarchar(stmt, 1, s) != duckdb.StateSuccess {
			panic("bind varchar")
		}
		duckdb.ClearBindings(stmt)
	}
}

func scnBindBlob(iters int) {
	db, conn := openMemoryDB()
	defer duckdb.Close(&db)
	defer duckdb.Disconnect(&conn)
	var stmt duckdb.PreparedStatement
	if duckdb.Prepare(conn, "SELECT $1", &stmt) != duckdb.StateSuccess {
		panic("prepare bind_blob")
	}
	defer duckdb.DestroyPrepare(&stmt)
	blob := make([]byte, 128)
	for j := range blob {
		blob[j] = byte(j)
	}
	for i := 0; i < iters; i++ {
		if duckdb.BindBlob(stmt, 1, blob) != duckdb.StateSuccess {
			panic("bind blob")
		}
		duckdb.ClearBindings(stmt)
	}
}

func scnCreateVarchar(iters int) {
	const s = "hello-duckdb-go-bindings"
	for i := 0; i < iters; i++ {
		v := duckdb.CreateVarchar(s)
		duckdb.DestroyValue(&v)
	}
}

func scnCreateEnum64(iters int) {
	names := make([]string, 64)
	for i := range names {
		names[i] = fmt.Sprintf("enum_val_%02d", i)
	}
	for i := 0; i < iters; i++ {
		lt := duckdb.CreateEnumType(names)
		duckdb.DestroyLogicalType(&lt)
	}
}

func scnValidUtf8_512(iters int) {
	buf := make([]byte, 512)
	for i := range buf {
		buf[i] = 'a'
	}
	for i := 0; i < iters; i++ {
		ed := duckdb.ValidUtf8Check(buf)
		duckdb.DestroyErrorData(&ed)
	}
}

func scnExecutePrepared(iters int) {
	db, conn := openMemoryDB()
	defer duckdb.Close(&db)
	defer duckdb.Disconnect(&conn)
	var stmt duckdb.PreparedStatement
	if duckdb.Prepare(conn, "SELECT 1 AS x", &stmt) != duckdb.StateSuccess {
		panic("prepare exec")
	}
	defer duckdb.DestroyPrepare(&stmt)
	for i := 0; i < iters; i++ {
		var res duckdb.Result
		if duckdb.ExecutePrepared(stmt, &res) != duckdb.StateSuccess {
			panic("execute prepared")
		}
		duckdb.DestroyResult(&res)
	}
}

func scnMixedPipeline(iters int) {
	db, conn := openMemoryDB()
	defer duckdb.Close(&db)
	defer duckdb.Disconnect(&conn)
	var stmt duckdb.PreparedStatement
	if duckdb.Prepare(conn, "SELECT $1::VARCHAR || '-' || $2::VARCHAR", &stmt) != duckdb.StateSuccess {
		panic("prepare mixed")
	}
	defer duckdb.DestroyPrepare(&stmt)
	a := "alpha-tag"
	b := "beta-tag-12345"
	for i := 0; i < iters; i++ {
		if duckdb.BindVarchar(stmt, 1, a) != duckdb.StateSuccess {
			panic("bind a")
		}
		if duckdb.BindVarchar(stmt, 2, b) != duckdb.StateSuccess {
			panic("bind b")
		}
		var res duckdb.Result
		if duckdb.ExecutePrepared(stmt, &res) != duckdb.StateSuccess {
			panic("execute mixed")
		}
		duckdb.DestroyResult(&res)
		duckdb.ClearBindings(stmt)
	}
}

func main() {
	variant := flag.String("variant", "unknown", "label printed in output (fork|upstream)")
	runs := flag.Int("runs", 5, "number of measured runs per scenario")
	warmup := flag.Int("warmup", 1, "warmup runs per scenario (not measured)")
	only := flag.String("only", "", "comma-separated list of scenarios to run (default: all)")
	openClose := flag.Int("open_close", 5_000, "iters for open_close scenario")
	prep := flag.Int("prepare_short", 200_000, "iters for prepare_short scenario")
	query := flag.Int("query_select1", 200_000, "iters for query_select1 scenario")
	bindV := flag.Int("bind_varchar", 2_000_000, "iters for bind_varchar scenario")
	bindB := flag.Int("bind_blob", 2_000_000, "iters for bind_blob scenario")
	cvarchar := flag.Int("create_varchar", 2_000_000, "iters for create_varchar scenario")
	cenum := flag.Int("create_enum64", 200_000, "iters for create_enum64 scenario")
	utf8c := flag.Int("valid_utf8_512", 2_000_000, "iters for valid_utf8_512 scenario")
	exec := flag.Int("execute_prepared", 200_000, "iters for execute_prepared scenario")
	mixed := flag.Int("mixed_pipeline", 100_000, "iters for mixed_pipeline scenario")
	flag.Parse()

	type scn struct {
		name  string
		iters int
		fn    func(int)
	}
	all := []scn{
		{"open_close", *openClose, scnOpenClose},
		{"prepare_short", *prep, scnPrepareShort},
		{"query_select1", *query, scnQuerySelect1},
		{"bind_varchar", *bindV, scnBindVarchar},
		{"bind_blob", *bindB, scnBindBlob},
		{"create_varchar", *cvarchar, scnCreateVarchar},
		{"create_enum64", *cenum, scnCreateEnum64},
		{"valid_utf8_512", *utf8c, scnValidUtf8_512},
		{"execute_prepared", *exec, scnExecutePrepared},
		{"mixed_pipeline", *mixed, scnMixedPipeline},
	}

	filter := map[string]bool{}
	if *only != "" {
		for _, n := range splitCSV(*only) {
			filter[n] = true
		}
	}

	report := benchReport{
		Variant:    *variant,
		Iterations: 0,
		Runs:       *runs,
		Warmup:     *warmup,
		GoVersion:  runtime.Version(),
		GoMaxProcs: runtime.GOMAXPROCS(0),
	}
	for _, s := range all {
		if len(filter) > 0 && !filter[s.name] {
			continue
		}
		fmt.Fprintf(os.Stderr, "[%s] running %-18s iters=%-9d runs=%d ...\n", *variant, s.name, s.iters, *runs)
		summary := runScenario(s.name, s.iters, *runs, *warmup, s.fn)
		report.Scenarios = append(report.Scenarios, summary)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(report); err != nil {
		fmt.Fprintln(os.Stderr, "encode:", err)
		os.Exit(2)
	}
}

func splitCSV(s string) []string {
	out := []string{}
	cur := ""
	for _, r := range s {
		if r == ',' {
			if cur != "" {
				out = append(out, cur)
				cur = ""
			}
			continue
		}
		cur += string(r)
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}
