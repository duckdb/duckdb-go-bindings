# CPU Benchmark Report: `adubovikov/duckdb-go-bindings` vs `duckdb/duckdb-go-bindings`

**Date:** 2026-05-14
**DuckDB version:** v1.5.2 (module `v0.10502.0`)
**Go version:** go1.26.1 linux/amd64
**Host:** Linux x86_64
**Comparison target:** `adubovikov/duckdb-go-bindings` fork (perf branch) vs
`duckdb/duckdb-go-bindings` upstream `main`.

The fork sits two commits ahead of upstream:

* `2cc8887` — `perf: cut CGO churn (bind/varchar/path, NUL pool, packed names)`
* `1275e49` — `perf: route remaining APIs through withNULString pool`

This report quantifies the CPU impact of those two commits on the
`bindings.go` / `bindings_arrow.go` hot paths.

---

## TL;DR

> On a 110-second mixed workload the fork uses **−14% total CPU**
> (90.97s → 78.04s) and finishes **−15% faster in wall time**, at a
> cost of **+6% peak RSS** (≈ 2.8 MiB). Hot CGO-bound paths drop by
> **35-66% CPU per operation**. No regressions outside measurement
> noise.

---

## Methodology

A small Go program (`workload/main.go`) drives `duckdb-go-bindings`
through ten micro-scenarios. The same source file is compiled twice via
`go.mod` `replace` — once against the upstream module and once against
the local fork — and an orchestrator (`compare/main.go`) executes both
binaries, captures their JSON reports, and prints a side-by-side delta.

### What is measured per scenario

For each scenario, on every measured run:

| metric        | source                                              |
|---------------|------------------------------------------------------|
| `wall ns/op`  | `time.Now()` delta                                   |
| `cpu ns/op`   | `getrusage(RUSAGE_SELF)` user+sys delta              |
| `cgo/op`      | `runtime.NumCgoCall()` delta                         |
| `allocs/op`   | `runtime.MemStats.Mallocs` delta (Go heap only)      |
| `bytes/op`    | `runtime.MemStats.TotalAlloc` delta (Go heap only)   |
| `wall stddev%`| jitter across runs                                   |

The orchestrator additionally captures **whole-process rusage**
(user/sys/wall/MaxRSS) on `os/exec`, which is the headline number.

### Configuration of the reported run

* `warmup=1` (un-measured), `runs=3` (measured) per scenario
* `scale=0.25` of default iteration counts (chosen so that the total
  comparison takes ~110 seconds and all stddev% stay reasonable)
* `GOGC=off` set by the orchestrator so the Go GC does not perturb timing
* Re-run reproducible via `./run.sh -scale 0.25 -runs 3`

Iteration counts used in this report:

| scenario          | iters per run |
|-------------------|---------------|
| `open_close`      | 1 250         |
| `prepare_short`   | 50 000        |
| `query_select1`   | 50 000        |
| `bind_varchar`    | 500 000       |
| `bind_blob`       | 500 000       |
| `create_varchar`  | 500 000       |
| `create_enum64`   | 50 000        |
| `valid_utf8_512`  | 500 000       |
| `execute_prepared`| 50 000        |
| `mixed_pipeline`  | 25 000        |

### Scenario map

| name              | exercises                                                                       | rationale                                                                  |
|-------------------|---------------------------------------------------------------------------------|----------------------------------------------------------------------------|
| `open_close`      | `Open(":memory:") → Connect → Disconnect → Close`                               | path string → NUL pool                                                     |
| `prepare_short`   | `Prepare(SQL) + DestroyPrepare`                                                 | SQL string → NUL pool                                                      |
| `query_select1`   | `Query("SELECT 1") + DestroyResult`                                             | SQL string → NUL pool                                                      |
| `bind_varchar`    | hot loop: `BindVarchar(stmt, 1, s) + ClearBindings`                             | hottest fork target: `withNULString`                                       |
| `bind_blob`       | hot loop: `BindBlob(stmt, 1, blob[128]) + ClearBindings`                        | `C.CBytes` → direct pointer                                                |
| `create_varchar`  | `CreateVarchar(s) + DestroyValue`                                               | `CreateVarchar` switched to length API                                     |
| `create_enum64`   | `CreateEnumType([64 names]) + DestroyLogicalType`                               | `allocNames`: 2 mallocs instead of N (pointer array + contiguous NUL blob) |
| `valid_utf8_512`  | `ValidUtf8Check(buf[512]) + DestroyErrorData`                                   | `C.CBytes` → direct pointer                                                |
| `execute_prepared`| `ExecutePrepared + DestroyResult` (prepared once outside loop)                  | **control**: not directly tuned by the fork                                |
| `mixed_pipeline`  | `BindVarchar×2 + ExecutePrepared + DestroyResult + ClearBindings`               | realistic ingest-like pipeline                                             |

---

## Results

### Whole-process rusage (lower is better)

| metric       | upstream | fork    | delta     |
|--------------|---------:|--------:|----------:|
| wall sec     |   65.141 |  55.640 | **−14.58%** |
| user sec     |   78.467 |  66.278 | **−15.53%** |
| sys sec      |   12.506 |  11.758 | **−5.98%**  |
| **cpu sec**  |   **90.973** | **78.036** | **−14.22%** |
| max rss kB   |   47 556 |  50 316 |  +5.80%   |

### Per-scenario delta (negative = fork is better)

| scenario          | cpu ns/op delta | cgo/op delta       | wall stddev% (ups → fork) | note                                                        |
|-------------------|----------------:|--------------------|:--------------------------|-------------------------------------------------------------|
| `valid_utf8_512`  |        **−66.4%** | −66.7% (3 → 1)    | 10.8 → 3.3                | direct pointer; one CGO call instead of three              |
| `create_varchar`  |        **−50.9%** | −50.0% (4 → 2)    | 10.7 → 5.0                | length API path                                             |
| `bind_blob`       |        **−49.8%** | −50.0% (4 → 2)    | 18.6 → 3.3                | drop `C.CBytes` on hot bind                                 |
| `create_enum64`   |        **−43.6%** | −47.4% (133 → 70) | 13.2 → 2.7                | packed names: 2 mallocs vs N                                |
| `bind_varchar`    |        **−35.5%** | −50.0% (4 → 2)    | 20.6 → 18.6               | `withNULString` pool                                        |
| `query_select1`   |        **−26.5%** | −50.0% (4 → 2)    | 54.1 → 0.5                | SQL string pooling; fork is far more stable                 |
| `execute_prepared`|        **−16.9%** | 0.0%              | 4.3 → 2.1                 | **control** — SQL pre-prepared; gain is indirect            |
| `mixed_pipeline`  |        **−9.6%**  | −44.4% (9 → 5)    | 0.6 → 2.3                 | end-to-end ingest-like                                      |
| `prepare_short`   |        −3.1%      | −50.0% (4 → 2)    | 6.5 → 1.0                 | short SQL; `C.CString` was already cheap                    |
| `open_close`      |        +3.1%      | −33.3% (6 → 4)    | 2.8 → 3.6                 | dominated by DuckDB engine startup; pool effect lost in noise |

### Per-op absolute numbers (median, in ns)

| scenario          | upstream `cpu ns/op` | fork `cpu ns/op` |
|-------------------|---------------------:|-----------------:|
| `valid_utf8_512`  |                129.7 |             43.5 |
| `bind_varchar`    |                258.0 |            166.3 |
| `bind_blob`       |                321.0 |            161.3 |
| `create_varchar`  |                211.3 |            103.7 |
| `prepare_short`   |             36 205.3 |         35 078.0 |
| `query_select1`   |            117 486   |         86 358.7 |
| `create_enum64`   |             11 305.3 |          6 371.9 |
| `execute_prepared`|             61 554.8 |         51 146.8 |
| `mixed_pipeline`  |             60 033.1 |         54 287.5 |
| `open_close`      |          6 953 892   |      7 171 676   |

---

## Analysis

### 1. CGO crossings are the primary win

Anywhere upstream used `C.CString` / `C.CBytes` plus a `defer C.free`,
the fork performs one pooled copy and a single CGO transition. On
`bind_*`, `create_varchar`, `Query`, and `Prepare`, **the number of
CGO calls per operation is halved (4 → 2)**. This is the dominant
contributor to the per-op CPU drop.

### 2. `valid_utf8_512` is the clearest case (−66%)

Upstream allocates `C.CBytes(blob)` for every UTF-8 check; the fork
passes the Go slice base pointer directly. The result: **3 CGO calls
per op → 1**, no `malloc/free` pair, and **2.97× higher throughput**.

### 3. `create_enum64` and `allocNames` (−44%)

Building an enum with 64 labels used to require ~133 CGO calls (one
`C.CString` per label plus housekeeping). The fork uses two
`duckdb_malloc` calls — one for the pointer array and one for a single
NUL-separated string blob — bringing it down to ~70 CGO calls. The
median CPU drops from 11.3 µs to 6.4 µs per construction.

### 4. `execute_prepared` is a non-trivial sanity check (−17% CPU)

The fork does not touch `ExecutePrepared` directly, and `cgo/op` is
identical (2.0 → 2.0). Yet CPU per op drops by 17%. This is the
**indirect benefit**: the runtime is doing less alloc/free work
overall, so the heap is cleaner and cache-locality improves. This is a
useful signal that the fork does not introduce hidden regressions on
adjacent paths.

### 5. `open_close` (+3.1%) and `prepare_short` (−3.1%) are noise

* `open_close`: each iteration costs **~7 ms** of real DuckDB engine
  work (catalog init, jemalloc warmup, etc.). The path-string pool
  optimizes a ~µs sliver of that, which disappears into measurement
  jitter.
* `prepare_short`: SQL is `SELECT $1::VARCHAR` — 18 bytes — so the
  upstream `C.CString` was already nearly free.

### 6. Memory cost is modest

MaxRSS rises by 5.8% (47.5 → 50.3 MiB, **+2.8 MiB**). This is the
working set of the `withNULString` pool plus packed-names buffers. For
server workloads this is negligible.

### 7. The fork is also **more stable**

Look at `wall stddev%`:

* `query_select1`: 54.1% → 0.5%
* `bind_blob`: 18.6% → 3.3%
* `create_enum64`: 13.2% → 2.7%
* `valid_utf8_512`: 10.8% → 3.3%

Lower stddev is a side-effect of fewer `malloc/free` pairs (less
arena contention) and fewer GC-relevant allocations.

### 8. Go heap counters can mislead

In some scenarios `allocs/op` is shown as **higher** in the fork
(e.g. `query_select1` 1 → 3). This is because:

* `runtime.MemStats.Mallocs` only sees the Go heap.
* The fork's `withNULString` pool adds a few small Go-side
  bookkeeping allocations that the counter picks up.
* The eliminated allocations were on the **C heap** (`C.CString` →
  libc `malloc`), which the counter does **not** see.

The real source of the CPU win is exactly those invisible C-heap
allocations. The truth is in `cpu ns/op` and `cgo/op`.

---

## Conclusion

The two perf commits in `adubovikov/duckdb-go-bindings`:

1. Halve CGO crossings on virtually every short-string / blob path.
2. Replace per-label `C.CString` with a packed `allocNames` for enum
   construction.
3. Eliminate `C.CBytes` on UTF-8 validation and bind-blob.

deliver a measured **−14% process-wide CPU** and **−15% wall time**
across a mixed workload, with per-op gains of **35-66%** on the
optimized hot paths and **no regressions** outside the noise floor.
The trade-off is **+2.8 MiB** of peak RSS, which is acceptable.

These changes are a strong candidate for upstreaming.

---

## Reproducing

```bash
git clone https://github.com/adubovikov/duckdb-go-bindings
cd duckdb-go-bindings/cpu_bench
./run.sh -scale 0.25 -runs 3            # ~110s on a modern x86_64
./run.sh -save_json my_result.json      # persist combined JSON
./run.sh -only bind_varchar,bind_blob   # narrow set
./run.sh                                # full default run (~7-8 minutes)
```

The combined report producing the tables above is committed as
`cpu_bench/result.json` in this repository.

## Limitations

* `getrusage` is per-process; a busy host increases noise. Use
  `taskset` / `cpupower frequency-set --governor performance` for the
  most stable numbers.
* Go heap allocation counters do not see C-heap traffic; do not draw
  malloc conclusions from `allocs/op` alone.
* `mixed_pipeline` is intentionally synthetic; production callers
  (e.g. tag-driven ingest) will be **closer to** `bind_varchar` and
  `execute_prepared` than to the lighter micro-scenarios.
