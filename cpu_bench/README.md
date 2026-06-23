# cpu_bench — CPU comparison: adubovikov fork vs duckdb upstream

This benchmark compares CPU/CGO/heap usage between two builds of
`github.com/duckdb/duckdb-go-bindings`:

* **upstream** — the official module at the version pinned in
  `upstream/go.mod` (currently `v0.10502.0`, DuckDB `v1.5.2`).
* **fork** — the local working tree of this repository (the
  `adubovikov/duckdb-go-bindings` fork that adds the `withNULString`
  pool, packed `CreateEnumType` names, and other CGO-churn reductions).

The same workload binary is compiled twice — once per module variant via
`go.mod replace` — and an orchestrator runs both, captures the JSON
reports plus process-level `rusage`, and prints a side-by-side delta.

## What it measures

For each scenario the workload records, per iteration:

* `wall ns/op`      — wall time
* `cpu ns/op`       — `(user + sys)` cpu time from `getrusage(RUSAGE_SELF)`
* `cgo calls/op`    — delta of `runtime.NumCgoCall()`
* `Go allocs/op`    — delta of `MemStats.Mallocs`
* `Go bytes/op`     — delta of `MemStats.TotalAlloc`

Each scenario runs `warmup` un-measured rounds and `runs` measured rounds;
medians and stddev are reported.

The orchestrator additionally captures the **whole-process** `rusage`
(user/sys/wall/MaxRSS) so you can also compare overall CPU spent.

## Scenarios

| name              | what it exercises                                                |
|-------------------|-------------------------------------------------------------------|
| `open_close`      | `Open(":memory:") + Connect + Disconnect + Close` (path NUL pool) |
| `prepare_short`   | `Prepare("SELECT $1::VARCHAR") + DestroyPrepare`                  |
| `query_select1`   | `Query("SELECT 1") + DestroyResult` (query string NUL pool)       |
| `bind_varchar`    | `BindVarchar(stmt,1,s) + ClearBindings` hot loop                  |
| `bind_blob`       | `BindBlob(stmt,1,blob) + ClearBindings` hot loop                  |
| `create_varchar`  | `CreateVarchar(s) + DestroyValue` (length API path)               |
| `create_enum64`   | `CreateEnumType([64 names]) + DestroyLogicalType` (packed names)  |
| `valid_utf8_512`  | `ValidUtf8Check(buf) + DestroyErrorData` (avoids `C.CBytes`)      |
| `execute_prepared`| `ExecutePrepared + DestroyResult` (control: not directly tuned)   |
| `mixed_pipeline`  | `BindVarchar*2 + ExecutePrepared + DestroyResult + ClearBindings` |

`open_close` is intentionally cheaper (`5_000` iters) because each
iteration involves real DuckDB startup work.

## Quick start

```bash
cd cpu_bench
./run.sh
```

For a fast smoke run (≈10% of default iterations):

```bash
./run.sh -scale 0.1 -warmup 1 -runs 3
```

Pick specific scenarios:

```bash
./run.sh -only bind_varchar,bind_blob,create_varchar -runs 5
```

Persist a combined JSON report:

```bash
./run.sh -save_json result.json
```

## Project layout

```
cpu_bench/
├── workload/main.go      # the actual workload (compiled into both variants)
├── upstream/             # go.mod: pulls upstream module
│   ├── go.mod
│   └── main.go           # symlink → ../workload/main.go
├── fork/                 # go.mod: replace → ../..  (the local fork)
│   ├── go.mod
│   └── main.go           # symlink → ../workload/main.go
├── compare/main.go       # orchestrator: builds & runs both, prints table
└── run.sh                # one-liner: build + build + compare
```

## Caveats

* CPU numbers come from `getrusage`; on heavily loaded boxes results
  jitter. Use `-runs` ≥ 5 and re-run if `wall stddev%` is large.
* Go heap counters (`allocs/op`, `bytes/op`) cover only the Go side. The
  C-side `malloc/free` traffic of the fork's `withNULString` pool will
  show up indirectly through `cpu ns/op` and `cgo/op`, not in
  `allocs/op`. Use `perf stat` / `jemalloc_stats` for malloc-level
  inspection if needed.
* `GOGC=off` is set during the run so GC events don't perturb timing.
* The fork's `lib/<platform>` prebuilt static libraries are referenced
  via `replace` in `fork/go.mod`, so you do not need to publish your
  own tags to bench locally.
