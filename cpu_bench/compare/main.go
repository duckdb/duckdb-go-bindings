// Compare runs the two bench binaries (fork and upstream) sequentially,
// captures their JSON reports and a process-level rusage summary, then prints
// a side-by-side comparison table.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"syscall"
	"time"
)

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

	WholeProcess processStats `json:"whole_process"`
}

type processStats struct {
	WallSec float64 `json:"wall_sec"`
	UserSec float64 `json:"user_sec"`
	SysSec  float64 `json:"sys_sec"`
	MaxRSSKB int64  `json:"max_rss_kb"`
}

func runBinary(label, binary string, args []string) (benchReport, error) {
	cmd := exec.Command(binary, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "GOGC=off")
	start := time.Now()
	err := cmd.Run()
	wall := time.Since(start)
	if err != nil {
		return benchReport{}, fmt.Errorf("%s: %w", label, err)
	}
	var rep benchReport
	if e := json.Unmarshal(out.Bytes(), &rep); e != nil {
		return benchReport{}, fmt.Errorf("%s: bad json: %w; raw=%s", label, e, out.String())
	}
	if cmd.ProcessState != nil {
		if ru, ok := cmd.ProcessState.SysUsage().(*syscall.Rusage); ok {
			rep.WholeProcess = processStats{
				WallSec: wall.Seconds(),
				UserSec: float64(ru.Utime.Sec) + float64(ru.Utime.Usec)/1e6,
				SysSec:  float64(ru.Stime.Sec) + float64(ru.Stime.Usec)/1e6,
				MaxRSSKB: ru.Maxrss,
			}
		}
	}
	rep.Variant = label
	return rep, nil
}

func fmtPct(a, b float64) string {
	if b == 0 {
		return "  n/a "
	}
	d := (a - b) / b * 100
	return fmt.Sprintf("%+6.2f%%", d)
}

func collectNames(reps ...benchReport) []string {
	seen := map[string]struct{}{}
	for _, r := range reps {
		for _, s := range r.Scenarios {
			seen[s.Name] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for n := range seen {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

func pick(rep benchReport, name string) (scenarioSummary, bool) {
	for _, s := range rep.Scenarios {
		if s.Name == name {
			return s, true
		}
	}
	return scenarioSummary{}, false
}

func printScenarioTable(fork, ups benchReport) {
	hdr := []string{
		"scenario", "metric", "upstream", "fork", "delta",
	}
	rows := [][]string{}
	names := collectNames(fork, ups)
	for _, n := range names {
		f, fok := pick(fork, n)
		u, uok := pick(ups, n)
		if !fok || !uok {
			rows = append(rows, []string{n, "MISSING", fmt.Sprint(uok), fmt.Sprint(fok), ""})
			continue
		}
		rows = append(rows, []string{n, "ns/op",       fmtFloat(u.NsPerOp),     fmtFloat(f.NsPerOp),     fmtPct(f.NsPerOp, u.NsPerOp)})
		rows = append(rows, []string{"", "cpu ns/op",   fmtFloat(u.CPUNsPerOp),  fmtFloat(f.CPUNsPerOp),  fmtPct(f.CPUNsPerOp, u.CPUNsPerOp)})
		rows = append(rows, []string{"", "cgo/op",      fmtFloat(u.CGOPerOp),    fmtFloat(f.CGOPerOp),    fmtPct(f.CGOPerOp, u.CGOPerOp)})
		rows = append(rows, []string{"", "allocs/op",   fmtFloat(u.GoAllocsOp),  fmtFloat(f.GoAllocsOp),  fmtPct(f.GoAllocsOp, u.GoAllocsOp)})
		rows = append(rows, []string{"", "bytes/op",    fmtFloat(u.GoBytesOp),   fmtFloat(f.GoBytesOp),   fmtPct(f.GoBytesOp, u.GoBytesOp)})
		rows = append(rows, []string{"", "wall stddev%", fmtFloat(safeDiv(u.WallNsStd, u.WallNsMean)*100), fmtFloat(safeDiv(f.WallNsStd, f.WallNsMean)*100), ""})
		rows = append(rows, []string{"", "", "", "", ""})
	}
	printAligned(hdr, rows)
}

func safeDiv(a, b float64) float64 {
	if b == 0 {
		return 0
	}
	return a / b
}

func fmtFloat(v float64) string {
	if v == 0 {
		return "0"
	}
	switch {
	case v >= 100000:
		return fmt.Sprintf("%.0f", v)
	case v >= 100:
		return fmt.Sprintf("%.1f", v)
	default:
		return fmt.Sprintf("%.3f", v)
	}
}

func printAligned(hdr []string, rows [][]string) {
	cols := len(hdr)
	widths := make([]int, cols)
	for i, h := range hdr {
		if len(h) > widths[i] {
			widths[i] = len(h)
		}
	}
	for _, r := range rows {
		for i, c := range r {
			if len(c) > widths[i] {
				widths[i] = len(c)
			}
		}
	}
	printRow := func(r []string) {
		parts := make([]string, cols)
		for i, c := range r {
			parts[i] = fmt.Sprintf("%-*s", widths[i], c)
		}
		fmt.Println("  " + strings.Join(parts, "  "))
	}
	printRow(hdr)
	sep := make([]string, cols)
	for i := range sep {
		sep[i] = strings.Repeat("-", widths[i])
	}
	printRow(sep)
	for _, r := range rows {
		printRow(r)
	}
}

func printWholeProcess(fork, ups benchReport) {
	rows := [][]string{
		{"wall sec", fmtFloat(ups.WholeProcess.WallSec), fmtFloat(fork.WholeProcess.WallSec), fmtPct(fork.WholeProcess.WallSec, ups.WholeProcess.WallSec)},
		{"user sec", fmtFloat(ups.WholeProcess.UserSec), fmtFloat(fork.WholeProcess.UserSec), fmtPct(fork.WholeProcess.UserSec, ups.WholeProcess.UserSec)},
		{"sys sec",  fmtFloat(ups.WholeProcess.SysSec),  fmtFloat(fork.WholeProcess.SysSec),  fmtPct(fork.WholeProcess.SysSec,  ups.WholeProcess.SysSec)},
		{"cpu sec",  fmtFloat(ups.WholeProcess.UserSec + ups.WholeProcess.SysSec), fmtFloat(fork.WholeProcess.UserSec + fork.WholeProcess.SysSec),
			fmtPct(fork.WholeProcess.UserSec+fork.WholeProcess.SysSec, ups.WholeProcess.UserSec+ups.WholeProcess.SysSec)},
		{"max rss kb", fmt.Sprintf("%d", ups.WholeProcess.MaxRSSKB), fmt.Sprintf("%d", fork.WholeProcess.MaxRSSKB),
			fmtPct(float64(fork.WholeProcess.MaxRSSKB), float64(ups.WholeProcess.MaxRSSKB))},
	}
	printAligned([]string{"metric", "upstream", "fork", "delta"}, rows)
}

func main() {
	forkBin := flag.String("fork", "./bench_fork", "path to fork bench binary")
	upsBin  := flag.String("upstream", "./bench_upstream", "path to upstream bench binary")
	runs    := flag.Int("runs", 5, "runs per scenario forwarded to bench")
	warmup  := flag.Int("warmup", 1, "warmup runs forwarded to bench")
	only    := flag.String("only", "", "limit scenarios (comma list) forwarded to bench")
	saveJSON := flag.String("save_json", "", "if set, write combined JSON report to this path")
	scale   := flag.Float64("scale", 1.0, "multiplier on default iters (forwarded to bench)")
	flag.Parse()

	common := []string{
		"-runs", fmt.Sprint(*runs),
		"-warmup", fmt.Sprint(*warmup),
	}
	if *only != "" {
		common = append(common, "-only", *only)
	}
	// Default iter sizes (must match bench defaults). We scale them here so users
	// can shrink runs quickly with -scale=0.1 for smoke tests.
	type pair struct{ flag string; def int }
	iterDefaults := []pair{
		{"-open_close", 5_000},
		{"-prepare_short", 200_000},
		{"-query_select1", 200_000},
		{"-bind_varchar", 2_000_000},
		{"-bind_blob", 2_000_000},
		{"-create_varchar", 2_000_000},
		{"-create_enum64", 200_000},
		{"-valid_utf8_512", 2_000_000},
		{"-execute_prepared", 200_000},
		{"-mixed_pipeline", 100_000},
	}
	for _, p := range iterDefaults {
		v := int(float64(p.def) * *scale)
		if v < 1 {
			v = 1
		}
		common = append(common, p.flag, fmt.Sprint(v))
	}

	fmt.Fprintln(os.Stderr, "==> running upstream bench:", *upsBin)
	ups, err := runBinary("upstream", *upsBin, append([]string{"-variant", "upstream"}, common...))
	if err != nil {
		fmt.Fprintln(os.Stderr, "upstream failed:", err)
		os.Exit(2)
	}
	fmt.Fprintln(os.Stderr, "==> running fork bench:", *forkBin)
	fork, err := runBinary("fork", *forkBin, append([]string{"-variant", "fork"}, common...))
	if err != nil {
		fmt.Fprintln(os.Stderr, "fork failed:", err)
		os.Exit(2)
	}

	fmt.Println()
	fmt.Println("== Whole-process rusage (lower = better) ==")
	printWholeProcess(fork, ups)
	fmt.Println()
	fmt.Println("== Per-scenario metrics (delta = fork vs upstream; negative = fork is better) ==")
	printScenarioTable(fork, ups)

	if *saveJSON != "" {
		combined := struct {
			Upstream benchReport `json:"upstream"`
			Fork     benchReport `json:"fork"`
		}{ups, fork}
		f, err := os.Create(*saveJSON)
		if err != nil {
			fmt.Fprintln(os.Stderr, "save_json:", err)
			os.Exit(2)
		}
		defer f.Close()
		enc := json.NewEncoder(f)
		enc.SetIndent("", "  ")
		if err := enc.Encode(combined); err != nil {
			fmt.Fprintln(os.Stderr, "save_json encode:", err)
			os.Exit(2)
		}
		fmt.Fprintln(os.Stderr, "wrote", *saveJSON)
	}
}
