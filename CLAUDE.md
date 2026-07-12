# go-time

Time semantics foundation library for Agent OS. Converts ambiguous human time expressions into precise, computable value objects. Hands off to stdlib `time.Time` / `time.Duration` for any formatting / display concern. Powers CLIs, services, and upstream agent systems.

**Module path**: `github.com/agentable/go-time`

## Commands

```bash
task test          # Run all tests with race detection
task lint          # Run golangci-lint v2 + go mod tidy check
task fmt           # Format Go code
task vet           # Run go vet
task verify        # Full verification: deps, fmt, vet, lint, test, vuln
task deps          # Download and tidy dependencies
task deps:update   # Update all dependencies
task clean         # Clean build artifacts
task vuln          # Run govulncheck
```

## Documentation Responsibilities

Keep the three documentation layers separate:

- `README.md` — human-facing usage guide: installation, quick start, API overview, runnable examples
- `SPECS/` — source of truth for contracts: parsing behavior, formatting rules, timezone semantics, error structure, integration boundaries
- `CLAUDE.md` / `AGENTS.md` — agent workflow, architecture rules, coding constraints, and doc-maintenance guidance

When updating documentation:

1. Update `README.md` only with current public usage
2. Update `SPECS/` only with rules that code can satisfy or violate
3. Update `CLAUDE.md` when workflow, architecture constraints, or doc-maintenance rules change

Do not duplicate contracts in `README.md`, and do not turn `SPECS/` into tutorials.

## Architecture

Three-layer architecture with strict unidirectional dependencies. Upper layers never pollute lower layers. **Formatting is not a layer of this library** — displaying values is the caller's concern, bridged via stdlib `time.Time` / `time.Duration`.

```text
github.com/agentable/go-time/
├── current.go          # Current-time helpers: Now / NowIn / TodayIn (no Today() — must specify zone)
├── options.go          # Parse options: WithInputLocale(language.Tag), WithZone, WithReference
├── instant.go          # Instant (absolute UTC; UnixSeconds/Millis/Nanos short forms; Add(Duration) only)
├── datetime.go         # DateTime (zoned local time; checked Add; AddPeriod→LocalResolution; .Clock()→Time)
├── local_datetime.go   # LocalDateTime (date + clock before zone resolution; Resolve(Zone)→candidates)
├── date.go             # Date (calendar date; checked Add(Period); unresolved until paired with Time + Zone)
├── time.go             # Time (clock time; unresolved until paired with Date + Zone)
├── duration.go         # Duration = type Duration time.Duration (no Day constant); .Decompose()→DurationComponents
├── period.go           # Period = struct{Years, Months, Days int32} (EOM clamp); fields exported directly, no Decompose
├── interval.go         # Interval (half-open [start, end); .StdRange()→(time.Time, time.Time))
├── zone.go             # Zone identity + stdlib Location bridge (IANA only in JSON)
├── duration_components.go # DurationComponents struct — Hours…Nanoseconds slots for external formatters
├── parse.go            # Layer 2: ISO 8601 / RFC 3339 parsing + ParseResult tagged-sum
├── parse_typed.go      # Layer 2: ParseInstant/ParseDateTime/ParseLocalDateTime/.../ParseInterval (8 typed parsers)
├── parse_impl.go       # Layer 2/3: parse dispatch — P{date}→Period, PT{time}→Duration, mixed→Invalid
├── parse_slash.go      # Layer 2: closed slash-locale policy; otherwise 0 invalid, 1 resolved, 2 ambiguous
├── errors.go           # ErrorCode + *TimeError + sentinel Err* instances
└── internal/
    ├── natural/        # Layer 2 helper: NL grammar to civil components (ar, en, hi, ja, ko, latin, zh; no zone lookup)
    └── zone/           # IANA data + DST projection + generated Windows→IANA mapping (`geniana/`, `genwindows/`)
```

**Layer dependency rules:**

- Layer 1 (value objects) — semantics and arithmetic use stdlib only; wire methods use the single `go-json-experiment/json` dependency. `zone.go` also imports `internal/zone` for static generated data.
- Layer 2 (`parse.go`, `internal/natural/`) — depends only on Layer 1 + stdlib + `golang.org/x/text/language`
- Layer 3 (arithmetic methods on value objects) — depends only on Layer 1
- User-visible API is `gotime.*` only — no internal dependencies leak
- **There is no formatting layer.** Any rendering happens outside this module via stdlib bridges (`.Std()` / `.Decompose()`)

## Agent Workflow

### Design Phase — Read SPECS First

Before designing or modifying any code, **read the relevant `SPECS/` documents first**. SPECS define domain model contracts, parsing rules, formatting rules, and architectural decisions. Do not invent new patterns or conventions that contradict SPECS.

**Workflow**:

1. Identify which SPECS are relevant to your task (see SPECS Index below)
2. Read those SPECS completely
3. Design your solution following SPECS constraints
4. If SPECS are unclear or incomplete, ask the user before proceeding

### Implementation Phase — Find 2 References First

Before writing implementation code, **find at least 2 relevant exemplars in `.references/`** to study their patterns, API design, and conventions. Browse the reference directories, read their source code, and adapt proven patterns rather than inventing from scratch.

**Workflow**:

1. Browse `.references/` (see References Guidance below)
2. Find 2+ projects relevant to your task
3. Study their implementation patterns
4. Adapt patterns to this project's conventions
5. If no relevant references exist, ask the user before inventing new patterns

## SPECS Index

Specification documents in [`SPECS/`](SPECS/) — system contracts, data formats, and design decisions:

| Spec | Topic |
|------|-------|
| [`00-overview.md`](SPECS/00-overview.md) | Project positioning, design principles, architecture layers, **Permanent Non-Goals** |
| [`10-domain-model.md`](SPECS/10-domain-model.md) | Core value objects (Duration/Period split), JSON schemas, **Wire Format Invariance** |
| [`20-parsing.md`](SPECS/20-parsing.md) | `Parse` + 8 typed `Parse*` functions, tagged-sum result, P/PT dispatch, ambiguity via Candidates |
| [`30-formatting.md`](SPECS/30-formatting.md) | Why this library doesn't format — stdlib bridge contract (`.Std()` / `.Decompose()`) |
| [`40-computation.md`](SPECS/40-computation.md) | Checked exact/calendar arithmetic, EOM clamp, and half-open interval operations |
| [`50-timezone.md`](SPECS/50-timezone.md) | IANA zones, deterministic JSON, DST nonexistent/duplicate handling |
| [`60-errors.md`](SPECS/60-errors.md) | Sentinel `Err*` + typed `*TimeError`, `ErrorCode` JSON metadata |
| [`70-api-surface.md`](SPECS/70-api-surface.md) | Public API surface, factory naming, short `Unix*` forms, `var UTC`, single `MustLoadZone` |
| [`80-integration.md`](SPECS/80-integration.md) | Integration boundaries for CLIs, services, and upstream systems |

When public behavior changes, update the relevant spec in the same work cycle.
If code and a spec disagree, treat the spec as stale and bring it back into sync immediately.

## References Guidance

Reference projects live in [`.references/`](.references/). Treat them as private study material, not public rationale. Pick relevant examples by capability:

- Functional API shape and tree-shakable organization
- Immutable value-object design
- Timezone data bundling and IANA rule handling
- Rich datetime semantics with duration/period separation
- Type-safe time handling and compile-time-friendly contracts

Do not cite exemplar names in SPECS, README, package docs, or public rationale.

## Design Philosophy

These nine principles override all SPECS and code. When a current API contradicts them, the API loses.

1. **One obvious way.** Every concept has exactly one constructor, one accessor, one verb.
2. **Names read as English.** `gotime.Now()`, `dt.In(zone)`, `iv.Contains(t)`.
3. **No surprise.** Functions that can fail return `error`. The only `Must*` is `MustLoadZone` (mirrors `regexp.MustCompile` for `var` initialization).
4. **Primitives, not products.** Parsing, value semantics, arithmetic, structured diagnostics — nothing else. No formatting/display, ambiguity UX, scheduling, persistence, CLI protocol.
5. **Immutable values, method receivers.** All value objects immutable; methods over package helpers when receiver is obvious.
6. **One concept per type.** `Duration` is exact nanoseconds. `Period` is calendar Y/M/D. `Interval` carries no formatting state.
7. **Stable wire format.** Every JSON shape is deterministic. Never recompute at marshal time from `time.Now()` or other ambient state.
8. **Errors compose with stdlib.** Sentinel `Err*` for `errors.Is`, typed `*TimeError` for `errors.As` — same pattern as `os.ErrNotExist` + `*fs.PathError`.
9. **Minimal dependency surface.** Public API uses only stdlib types + `golang.org/x/text/language.Tag`. go-time never imports display i18n, CLDR, message-format, or formatting packages. Controlled input phrase grammars are parsing data, not a display layer.

If a coding decision violates one of these, the principle wins and the decision is wrong.

Operational corollaries:
- **KISS** — One `Parse()` handles all inputs. No formatters in this package — display is the caller's problem.
- **DRY** — Each concern lives in exactly one package. Display locale data lives elsewhere; controlled input phrase grammars stay small and parser-owned until an adapter boundary is proven.
- **YAGNI** — Not a calendar, scheduler, cron, recurrence, or formatting library. Permanently out of scope.
- **Simplicity as art** — `gotime.Now()` → `Instant` (zero config). `gotime.TodayIn(z)` → `Date` (zone is mandatory — a calendar date with no zone is undefined).
- **Never:** accidental complexity, feature gravity, abstraction theater, configurability cope, importing formatters.

## Coding Rules

### Must Follow

- Go 1.26 — use modern language features where they simplify code
- Follow [Google Go Best Practices](https://google.github.io/go-style/best-practices)
- Follow [Google Go Style Decisions](https://google.github.io/go-style/decisions)
- KISS/DRY/YAGNI — no premature abstractions, no unused features, no duplicated logic
- Explicit error handling — return errors, wrap with context via `fmt.Errorf("%w")`
- All value objects are immutable — all operations return new values, no setters
- Use concrete time types (`Instant`, `DateTime`, `Date`, `Time`, `Duration`, `Period`, `Interval`, `Zone`) — never `string` or `any` for time values
- Public API never exposes locale / formatter / i18n types — accept `language.Tag` only for parser hints
- Every error must include an actionable Hint — tell the user how to fix it
- **Calendar math is `AddPeriod(Period)`; exact math is `Add(Duration)`** — `dt.Add(gotime.Months(1))` must fail at compile time. The type system is load-bearing here.
- `Duration` is `type Duration time.Duration` — `Nanosecond` through `Hour` are typed constants supporting `5 * gotime.Minute` arithmetic. No `Day` constant (24h is `24 * Hour`; calendar day is `Days(n)` on Period). No `Hours(n)` / `Days(n float64)` constructors.
- `Period` fields (`Years`, `Months`, `Days`) are exported — literal initialization is the canonical form. Constructors `Years(n)` / `Months(n)` / `Days(n)` are sugar.
- `Period.Add`, `Sub`, `Negate`, and `Abs` are checked and return `ErrOverflow`; calendar component arithmetic must never wrap.
- `Period` month/year add applies end-of-month clamping (Jan 31 + Months(1) = Feb 28/29). Never overflow.
- Intervals are half-open `[start, end)` — `Contains` excludes end, `Overlaps` excludes touching endpoints, use `Adjacent` for boundary detection
- `Interval.Expand(before, after)` returns `(Interval, error)` — reject negative expansion durations and preserve the same `end >= start` invariant as constructors.
- `Interval` carries no zone field — projection zone belongs to the rendering layer (which is outside this module)
- Use `ResolveZone` for fuzzy timezone resolution (Windows names and case-insensitive IANA names) — use `LoadZone` for strict IANA-only. Fixed offsets are not zones; RFC3339 numeric offsets parse to `Instant`.
- Treat `internal/zone/catalog.go` and `internal/zone/windows.go` as generated artifacts. Regenerate Windows mappings only from the CLDR release pinned in `SPECS/50-timezone.md`; never hand-edit or canonicalize its territory `001` targets.
- `.Std()` returns stdlib types (`time.Time` / `time.Duration`); `.Clock()` returns a `Time`; `Duration.Decompose()` returns `DurationComponents` (clock slots only). Naming is load-bearing: stdlib vs. clock vs. structured slot. `Period` has no `Decompose` — read `p.Years` / `p.Months` / `p.Days` directly (exported fields), no parallel struct.
- `Zone.Location()` is total — the zero `Zone` falls back to `UTC`; do not reintroduce parallel fallback helpers
- Parse option presence is explicit: `WithZone(Zone{})` means UTC and `WithReference(Instant{})` means the Go zero instant; never infer presence with `IsZero`.
- Relative natural dates/datetimes require both `WithReference` and `WithZone`; formal floating datetimes may omit `WithZone` and remain `LocalDateTime`.
- `internal/natural` receives an already-projected civil reference and returns civil components only; `LocalDateTime.Resolve` at the gotime boundary is the sole owner of natural datetime zone resolution.
- `Zone.MarshalJSON` outputs only `{"kind":"zone","id":"..."}` and normalizes zero `Zone` to `UTC` — never call `time.Now()` during marshal. Time-dependent offset and abbreviation projection belongs to stdlib `time.Time.Zone`.
- `ParseResult` accessors are comma-ok (`Instant() (Instant, bool)` etc.) — never silently return zero values when `Kind` doesn't match
- `ParseResult` has no public `Value() any` escape hatch — dispatch unknown input with `Status`, `Kind`, and comma-ok accessors.
- `ParseResult.HasZone` indicates whether the input explicitly included timezone/offset information — use this to detect floating times
- `errors.Is(err, gotime.ErrAmbiguousDate)` for control flow; `errors.As(err, &te)` for detail extraction. `*TimeError` unwraps its `Err` sentinel; `Code` is JSON/log metadata. Typed parsers map DST duplicate local-time ambiguity to `ErrDuplicateTime`, not generic date ambiguity.
- `Code*` is `ErrorCode` (string metadata); `Err*` is a sentinel `error`. Never mix prefixes.
- `MustLoadZone` is the only public `Must*`. Never add `MustParse`, `MustNewInterval`, etc.
- `Parse("P{date-only}")` (e.g. `P1Y`, `P5D`, `P2W`) routes to `KindPeriod`. `Parse("PT{time-only}")` routes to `KindDuration`. `Parse("P{date}T{time}")` returns `StatusInvalid` + `CodeInvalidFormat` — no single type can carry both halves.
- Duration / Period JSON contain **only** `{"kind":..., "iso":...}`. No `components` / `years` / `months` / `days` fields on the wire. Run `Duration.Decompose()` or read `Period` struct fields at the call site.
- `Duration.ISO8601()` uses canonical decimal seconds for sub-second precision; scientific notation is not accepted on the wire.
- Parse ambiguity (slash-date with no locale, DST fall-back) returns `StatusAmbiguous` with `Candidates`. The caller decides — there is no `WithStrategy` knob.
- `Duration.String()` matches `time.Duration.String()` byte-for-byte. Stringer is a stdlib contract; do not invent variants like `"1h30m"` that drop the trailing `0s`.
- Selection / clamping uses `Compare` + `slices.MinFunc` / `slices.MaxFunc`. There are no `Min` / `Max` / `Clamp` methods on any value object — they would either be asymmetric or balloon the API surface.

### Forbidden

**Top directive — the 10-year contract:**

- **go-time MUST NEVER import any display i18n / CLDR / message-format / formatting package.** No specific implementation or successor is exempt. The only language-related dependency permitted is `golang.org/x/text/language` (BCP-47 tag type, no display data).
- **go-time MUST NEVER expose a formatter type** (`DateTimeFormat`, `RelativeTimeFormat`, `DurationFormat`), a `Locale` type, a `HourCycle` / `Calendar` enum, or a `Style` enum.
- **go-time MUST NEVER ship display-locale JSON / YAML / template / CLDR data** in this repository.
- **No value-object method takes a formatter parameter.** `dt.Format(f)`, `d.Render(loc)`, etc. are banned. Rendering happens via `f.Format(dt.Std())` at the call site, in code the caller wrote.

**Other constraints:**

- No public panic except `MustLoadZone` (matching `regexp.MustCompile` precedent — for `var`/`init()` with source-code-constant IDs only)
- No premature abstraction — three similar lines are better than a helper used once
- No feature creep — only implement what's currently needed
- No merging `Instant` and `DateTime` into one type — they are semantically distinct
- No `Instant + Period` (Instant has no calendar view); no `Date + Duration` (Date has no time component); no `DateTime + DateTime`. Compile-time enforced.
- No coupling `Zone` and language — orthogonal dimensions, independently configured
- No `WithLocale` (ambiguous) — `WithInputLocale(language.Tag)` is the only language-aware option
- No `WithZoneID` — fuzzy resolution belongs at the call site (`ResolveZone`), not buried in an option
- No `WithStrategy` / `Strategy` enum — ambiguity surfaces via `ParseResult.Candidates`; callers pick
- No `gotime.Today()` zero-arg helper — a calendar date with no zone is undefined; require `TodayIn(z)`
- No `gotime.Day` Duration constant — calendar day is `Days(n)` (Period), 24h is `24 * Hour` (Duration); the name collision is a footgun
- No `Min` / `Max` / `Clamp` methods — selection composes with `slices.MinFunc(xs, gotime.Instant.Compare)` etc.
- No `StartOf*` / `EndOf*` family — half-open intervals don't need them; explicit construction is 3 lines and avoids the "does EndOfDay include the last nanosecond?" trap
- No `Duration.InDays()` — Days is a Period concept; if a caller wants 24-hour groups it can do `d.InHours() / 24`
- No JSON wire field that is also derivable from another field in the same payload — every value object's JSON has exactly one source of truth (`iso` for Duration/Period, etc.)
- No global mutable defaults (default zone, default language)
- No documentation masquerading as code — specs are the SSOT; code executes rules, not redescribes them
- No working around dependency bugs — if a bug or limitation is in a dependency library, do NOT bypass it. Create a report file in `reports/` (see Dependency Issue Reporting below)
- No new `encoding/json` imports — use `github.com/go-json-experiment/json` exclusively
- No reminder/alarm/event/cron/RRULE/business-calendar/astronomical types — permanent ban

### Domain Patterns

See SPECS/ for detailed patterns:

- [SPECS/10-domain-model.md](SPECS/10-domain-model.md) — Value objects, JSON schemas, type operations, immutability rules
- [SPECS/20-parsing.md](SPECS/20-parsing.md) — Three-status parse result model
- [SPECS/30-formatting.md](SPECS/30-formatting.md) — Why this library doesn't format; stdlib bridge contract
- [SPECS/40-computation.md](SPECS/40-computation.md) — Calendar vs exact math, end-of-month clamping, half-open interval operations
- [SPECS/60-errors.md](SPECS/60-errors.md) — Error codes, TimeError structure

## Testing

- Table-driven tests for value objects — cover all type operations
- Parse layer: positive + negative cases, at least one test per error code
- NL layer: each language x each time expression pattern
- Calendar math tests: end-of-month clamping, DST boundary preservation
- JSON round-trip tests: `Marshal → Unmarshal → Marshal` byte-identical
- Run specific tests: `go test -race -run TestName ./...`

## Dependencies

| Dependency | Purpose |
|------------|---------|
| `github.com/go-json-experiment/json` | Stable JSON serialization via jsonv2 |
| `github.com/google/go-cmp` | Test diffs only |
| `golang.org/x/text` | `language.Tag` for `WithInputLocale` — BCP-47 type only, no CLDR/display data |

**No other dependencies are permitted in `go.mod`.** Any addition that pulls in display i18n, CLDR, message-format, or display-locale data is forbidden by the top directive in Forbidden above.

## Error Handling

Sentinel + typed struct hybrid (`os.ErrNotExist` + `*fs.PathError` pattern). Go system errors use `error`; parse semantic states use `ParseResult.Status` (`StatusResolved` / `StatusAmbiguous` / `StatusInvalid`); typed `Parse*` functions translate `StatusAmbiguous`/`StatusInvalid` to a returned `*TimeError`.

`ErrorCode` constants (typed string, prefix `Code*`):
`CodeEmptyInput`, `CodeInvalidFormat`, `CodeInvalidDate`, `CodeInvalidTime`, `CodeInvalidDuration`, `CodeInvalidPeriod`, `CodeInvalidZone`, `CodeAmbiguousDate`, `CodeNonexistentTime`, `CodeDuplicateTime`, `CodeIntervalReversed`, `CodeIntervalsDisjoint`, `CodeUnparseable`, `CodeOverflow`, `CodeIncompatibleTypes`.

Sentinel `*TimeError` instances (one per code, prefix `Err*`):
`ErrEmptyInput`, `ErrInvalidFormat`, `ErrInvalidDate`, `ErrInvalidTime`, `ErrInvalidDuration`, `ErrInvalidPeriod`, `ErrInvalidZone`, `ErrAmbiguousDate`, `ErrNonexistentTime`, `ErrDuplicateTime`, `ErrIntervalReversed`, `ErrIntervalsDisjoint`, `ErrUnparseable`, `ErrOverflow`, `ErrIncompatibleTypes`.

Pattern:

```go
if errors.Is(err, gotime.ErrAmbiguousDate) { /* control flow */ }

var te *gotime.TimeError
if errors.As(err, &te) { log.Printf("code=%s hint=%s", te.Code, te.Hint) }
```

`*TimeError` unwraps its `Err` sentinel, so `errors.Is` follows the standard unwrap chain; `Code` does not drive matching.

## Linting

golangci-lint v2. Config in `.golangci.yml`. Includes: errorlint, exhaustive, gocritic, gosec, revive, and 15+ other linters.

## CI

GitHub Actions (`ci.yml`): test + lint on push/PR to main. Separate security job runs `govulncheck`.

## Pre-commit Hooks

Lefthook with parallel hooks: trailing-whitespace, gitleaks (secret detection), lint (`task lint`), test (`task test`), and yamllint.

## Dependency Issue Reporting

When you encounter a bug, limitation, or unexpected behavior in a dependency library:

1. **Do NOT** work around it by reimplementing the dependency's functionality
2. **Do NOT** skip or ignore the dependency and write your own version
3. **Do** create a report file: `reports/<dependency-name>.md`
4. **Do** include in the report:
   - Dependency name and version
   - Problem description (what went wrong)
   - Trigger scenario (what you were doing when you hit it)
   - Expected behavior vs actual behavior
   - Relevant error messages or stack traces
   - Workaround suggestion (if any, without implementing it)
5. **Do** continue with other tasks that don't depend on the broken functionality

The `reports/` directory is checked by team members after each work cycle. Reports are routed to the appropriate dependency maintainer for resolution.

## Monorepo Context

This package is part of a Go monorepo. See [root CLAUDE.md](../CLAUDE.md) for shared conventions.

## Agent Skills

Common implementation skills in `.agents/skills/`:

| Skill | When to Use |
|-------|------------|
| [agent-md-writing](.agents/skills/agent-md-writing/) | Regenerating `CLAUDE.md` and refreshing the `AGENTS.md` symlink |
| [readme-writing](.agents/skills/readme-writing/) | Regenerating `README.md` from the current public API |
| [golangci-linting](.agents/skills/golangci-linting/) | Setting up or running golangci-lint v2, fixing lint errors, configuring linters |
| [modernizing](.agents/skills/modernizing/) | Adopting Go 1.20-1.26 new features — generics, iterators, error handling, stdlib collections |
| [committing](.agents/skills/committing/) | Creating conventional commit messages for Go packages |
| [releasing](.agents/skills/releasing/) | Releasing a Go package — semantic versioning, tagging, dependency upgrades |
| [code-simplifying](.agents/skills/code-simplifying/) | Refining recently written Go code for clarity and consistency without changing functionality |
| [go-best-practices](.agents/skills/go-best-practices/) | Applying Google Go style guide — naming, error handling, interfaces, concurrency |
| [tdd-implementing](.agents/skills/tdd-implementing/) | Implementing features with strict TDD red-green-refactor cycles |
| [tdd-planning](.agents/skills/tdd-planning/) | Planning TDD implementation for features from specs |
| [spec-reviewing](.agents/skills/spec-reviewing/) | Reviewing SPECS for completeness and consistency |
| [code-refactoring](.agents/skills/code-refactoring/) | Architecture review and pragmatic refactoring |
| [architecture-audit](.agents/skills/architecture-audit/) | Full-codebase health checks, dependency analysis, layer boundary validation |

Design skills in `.agents/skills/`:

| Skill | When to Use |
|-------|------------|
| [golang-design-guide](.agents/skills/golang-design-guide/) | Designing Go libraries — type classification, design philosophy, API patterns, error handling strategy |

## License

This software is licensed under the **MIT License**.
See the [LICENSE](./LICENSE) file for full terms.
