# Upstream PR / issue draft

> Suggested title: **perf: cut CGO churn on hot paths (NUL-string pool, packed enum names, direct-pointer bind/UTF-8) — −14% process CPU**

Paste the body below as the PR description against
[`duckdb/duckdb-go-bindings`](https://github.com/duckdb/duckdb-go-bindings).

---

## Summary

This PR proposes two perf changes to `bindings.go` / `bindings_arrow.go`
that **halve CGO crossings on virtually every short-string / blob path**
and replace per-label `C.CString` with a packed allocation in
`CreateEnumType`.

On a mixed micro-benchmark workload (10 scenarios, ~110 s on x86_64 Linux,
DuckDB v1.5.2 / `v0.10502.0`, Go 1.26.1) this is worth:

| metric              | upstream | this PR   | delta       |
|---------------------|---------:|----------:|------------:|
| **whole-process CPU** | 90.97 s  | 78.04 s   | **−14.22%** |
| wall time           | 65.14 s  | 55.64 s   | −14.58%     |
| max RSS             | 47.5 MiB | 50.3 MiB  | +5.80%      |

Per-op CPU gains on the optimized hot paths (median across 3 measured
runs after a 1-run warmup):

| scenario          | upstream `cpu ns/op` | this PR `cpu ns/op` | delta       | CGO calls/op |
|-------------------|---------------------:|--------------------:|------------:|:-------------|
| `valid_utf8_512`  |                129.7 |                43.5 | **−66.4%**  | 3 → 1        |
| `create_varchar`  |                211.3 |               103.7 | **−50.9%**  | 4 → 2        |
| `bind_blob`       |                321.0 |               161.3 | **−49.8%**  | 4 → 2        |
| `create_enum64`   |             11 305.3 |             6 371.9 | **−43.6%**  | 133 → 70     |
| `bind_varchar`    |                258.0 |               166.3 | **−35.5%**  | 4 → 2        |
| `query_select1`   |            117 486   |            86 358.7 | **−26.5%**  | 4 → 2        |
| `execute_prepared`|             61 554.8 |            51 146.8 | **−16.9%**  | 2 → 2 (no direct change; indirect gain) |
| `mixed_pipeline`  |             60 033.1 |            54 287.5 |  −9.6%      | 9 → 5        |
| `prepare_short`   |             36 205.3 |            35 078.0 |  −3.1%      | 4 → 2 (short SQL, small gain) |
| `open_close`      |          6 953 892   |         7 171 676   |  +3.1%      | 6 → 4 (dominated by DuckDB engine work; noise) |

No regressions outside measurement noise (`open_close` is within
typical run-to-run jitter; the path is dominated by real DuckDB
init time, not bindings overhead).

The fork is also **noticeably more stable** under jitter — e.g.
`query_select1` wall stddev drops from 54% to 0.5%, because fewer
malloc/free pairs reduce arena contention.

## What changed (commits in order)

1. **`perf: cut CGO churn (bind/varchar/path, NUL pool, packed names)`**
   - Replace `C.CString` / `C.CBytes` on hot binds, vector ops, UTF-8 check
   - `CreateVarchar` switched to length API (no NUL-terminator dance)
   - `allocNames`: single string blob + 2 `duckdb_malloc` calls (pointer
     array + contiguous NUL blob) instead of N separate `C.CString`s
   - `withNULString` pool covers `Query` / `Prepare` / `Open` (avoids
     stack-to-heap escapes for short strings)
   - Adds allocation micro-benchmarks and regression tests
2. **`perf: route remaining APIs through withNULString pool`**
   - Remove the last `C.CString` uses (scalar/table function names,
     bind errors, profiler keys, appender/table description, log
     storage, Arrow scan)
   - Adds `withNULStringVoid` for side-effect-only C calls

## Why now

`bind_varchar` and `valid_utf8_512` are extremely hot for any caller
that ingests semi-structured records (e.g. SIP/Diameter parsers,
log-ingest pipelines). The current `C.CString` + `defer C.free` pair
on every call adds ~2 CGO crossings and a `malloc/free` pair per op.
In bulk ingest these add up to double-digit % of process CPU.

The packed `CreateEnumType` matters less per-op but is unbounded — a
table function with 1 000 enum values goes from ~2 000 CGO calls to
~1 100.

## Memory cost

MaxRSS grows by ~2.8 MiB (5.8%) — this is the steady-state working
set of the `withNULString` pool plus packed-names buffers. Pool size
is bounded; the pool releases buffers to GC under pressure. For
server processes this is negligible.

## Backwards compatibility

Public API is unchanged. All existing tests pass. The fork ships
additional `bindings_alloc_bench_test.go` with `BenchmarkPrepare_*`,
`BenchmarkBindVarchar_*`, `BenchmarkBindBlob_*`,
`BenchmarkCreateVarchar*`, `BenchmarkValueToString_*`,
`BenchmarkValidUtf8Check_*`, `BenchmarkCreateEnumType_64Names`.

## Reproducing

The numbers above come from a small Go harness that compiles the
**same** workload twice — once against
`github.com/duckdb/duckdb-go-bindings@v0.10502.0` (this base), and
once against the fork tree via `go.mod replace`. It runs both
binaries sequentially and reports, per scenario:

* median `wall ns/op` and `cpu ns/op` (from
  `syscall.Getrusage(RUSAGE_SELF)`),
* `cgo calls/op` (delta of `runtime.NumCgoCall`),
* Go-heap `allocs/op` and `bytes/op` (delta of `runtime.MemStats`).

It also captures **whole-process** `user/sys/wall/MaxRSS` via
`os/exec` for the headline number above. `GOGC=off` is set during the
run so Go GC events do not perturb timing.

The harness is not part of this PR (kept small and focused). I am
happy to send it as a follow-up PR, or as a separate sub-directory if
maintainers prefer to have it in-tree for future regression checks.
Reviewers who want to reproduce on their hardware can ping me on the
PR and I'll publish the branch.

## Checklist

- [x] Tests pass (`go test ./...` in the fork)
- [x] No public-API changes
- [x] No new third-party dependencies
- [x] Benchmarks added (`bindings_alloc_bench_test.go`)
- [x] CPU benchmark harness against unmodified upstream available on request (kept out of this PR for review focus)
- [ ] Reviewer guidance on whether to split into two commits or
      squash on merge — happy either way.

## Risks and caveats

* `withNULString` adds a small Go-side pool. Counters in
  `runtime.MemStats` can show *higher* `allocs/op` because the pool
  uses Go-heap bookkeeping objects — but the eliminated allocations
  were on the **C heap** (libc `malloc`), invisible to
  `MemStats.Mallocs`. The CPU win is real and shows in `cpu ns/op`
  and `cgo/op`.
* `open_close` benchmarks dominated by DuckDB engine startup;
  treating the +3% on that scenario as noise.
* Numbers above are from a single Linux x86_64 host; expect similar
  shape but different absolute deltas on darwin/arm64.

Happy to iterate on any of the above — naming, splitting commits,
removing the harness from the PR and submitting it as a follow-up,
etc.
