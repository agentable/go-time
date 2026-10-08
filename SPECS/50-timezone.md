# Time Zones

## Overview

go-time treats IANA timezone IDs as the canonical timezone identity and uses stdlib `time.Location` for projection. Offsets are input aids; abbreviations are display metadata, not durable identity.

## Zone API

```go
var UTC Zone

func LoadZone(id string) (Zone, error)
func MustLoadZone(id string) Zone
func ResolveZone(id string) (Zone, error)
func Zones() []string
func ZoneCatalogVersion() string
```

- `LoadZone` is strict IANA lookup from TZif rule data; `Local` is
  rejected by both loaders and by Zone/DateTime JSON decoders with `ErrInvalidZone`.
- `MustLoadZone` is only for source-code constants in `var` or `init` paths.
- `ResolveZone` accepts real-world zone names: exact IANA, case-insensitive IANA, Windows names, and legacy aliases present in the rule data.
- `ResolveZone` does not resolve timezone abbreviations. Abbreviations are
  point-in-time display metadata available through stdlib `time.Time.Zone`.
- `Zones` returns a sorted, cloned, caller-owned copy of the generated IANA
  identifier catalog, not every loadable IANA link. Do not use catalog membership
  as a loader allowlist. Mutating it does not affect the internal catalog or a
  later call. `ZoneCatalogVersion` returns the IANA tzdb version used to
  generate that catalog; it does not describe the transition-rule snapshot held by a Zone.
- `UTC` is the only predefined zone. Treat it as a named read-only value for
  reading, comparison, and arguments. Assigning to the exported variable is
  not a configuration mechanism and does not change zero-Zone semantics or a
  process-wide default. Host-local discovery and configuration
  belong to applications because process environment is not a durable zone
  identity.

Use `LoadZone` for stable configuration and persisted canonical IDs. Use `ResolveZone` for CLI args, forms, migration input, and compatibility paths.

## Zone Identity

```go
zone.ID()
zone.Location()
zone.String()
zone.Equal(other)
zone.IsZero()
```

`Zone.Location()` is total: the zero zone returns `time.UTC`.

The zero Zone has UTC value semantics across `ID`, `String`, `Equal`,
`Location`, and JSON. `IsZero` reports only whether the Go representation is
the zero value; parse option presence is tracked separately.

`Zone.MarshalJSON` is deterministic:

```json
{"kind":"zone","id":"Asia/Tokyo"}
```

It never emits offset, abbreviation, DST flags, or any field that would require a reference instant. A zero `Zone` marshals as `{"kind":"zone","id":"UTC"}` to match its total UTC projection behavior. Fixed UTC offsets are not zone identities; `ResolveZone("+08:00")` and `ResolveZone("UTC+8")` return `ErrInvalidZone`, and marshaling an internally malformed fixed-offset `Zone` returns `ErrInvalidZone` instead of emitting `{"kind":"zone","id":"+08:00"}`.

Encoding an already-loaded Zone or DateTime uses its held rules and does not
reload the zone from the environment. Replacing or deleting rule files cannot
change that value's encoding. Decoding loads the named rules at decode time;
the wire format does not promise a fixed tzdb snapshot across environments.
Civil and UTC wire domain checks still apply.

Each Zone holds a single rule snapshot: its `time.Location` and complete set of
possible UTC offsets are derived from the same TZif bytes. Loads search the
`ZONEINFO` directory or ZIP, conventional Unix zoneinfo directories, then the
bundled runtime archive. `ZONEINFO` is captured on first load. Platform-specific packed stores are not read; deployment
can select a directory or ZIP explicitly. Missing or invalid entries fall through
to later sources, and an unknown identity returns `ErrInvalidZone`. The bundled
archive's source and checksum are maintained in `internal/zone/zoneinfo.md`.

RFC 3339 values with numeric offsets parse to `Instant`. The offset is syntax for an absolute moment, not a persisted zone identity.

## Contract Decisions

### Identity Is An IANA Zone

- **Decision**: Persisted zones are IANA identifiers only. Numeric offsets identify instants in timestamp syntax, not reusable zone identities.
- **Why**: A zone carries historical and future transition rules. A fixed offset cannot answer local-time projection questions.
- **Rejected**: Persisting `+08:00` as `Zone`, guessing a zone from an offset, and accepting abbreviations as identity.
- **Contract Impact**: Offset-bearing RFC 3339 input resolves to `Instant`; `Zone` JSON remains `{"kind":"zone","id":"..."}`.

### Point-in-Time Display Facts Stay in the Stdlib

- **Decision**: callers obtain abbreviation and offset seconds from
  `instant.Std().In(zone.Location()).Zone()`.
- **Why**: `time.Time.Zone` already owns this projection. A parallel snapshot
  DTO or formatted offset string would duplicate stdlib behavior and pull
  display concerns into the semantic core.
- **Rejected**: parallel display DTOs, convenience offset/abbreviation methods,
  formatted offset strings, and heuristic DST flags.
- **Contract Impact**: DST gaps and overlaps remain semantic results from
  `LocalDateTime.Resolve`; display projection begins after conversion to
  stdlib `time.Time`.

### Transition Enumeration Is Not a go-time Contract

- **Decision**: go-time does not expose timezone transition enumeration or
  observance types. `Zone.Location()` bridges point-in-time projection and
  local-time resolution; it does not promise a complete transition sequence.
- **Why**: Protocols that serialize timezone rules own their data provenance,
  truncation window, and observance model. Those concerns do not belong in the
  semantic foundation.
- **Rejected**: `Zone.Transitions`, sampling or binary-search discovery, and a
  claim that repeatedly advancing stdlib `time.Time.ZoneBounds` completely
  enumerates runtime rules.
- **Contract Impact**: transition tests use a focused corpus to verify local
  projection behavior. Protocol libraries obtain authoritative transition data
  through their own integration boundary.

### Windows Name Mapping Is Generated

- **Decision**: Windows timezone names map from Unicode CLDR `release-48-1`
  (`225136a3bd3eff573f3d64c8fa25d6f9afa974ad`), using only territory `001`
  defaults. The checked-in `internal/zone/windows.go` is generated by
  `internal/zone/genwindows`.
- **Why**: an exact source and deterministic generator make mapping changes
  reviewable and prevent hand-maintained drift.
- **Rejected**: runtime CLDR/XML dependencies, network lookup, target-ID
  canonicalization, and rewriting valid backward-compatible IANA links.
- **Contract Impact**: `ResolveZone` combines static name mappings with the shared TZif
  snapshot loader at runtime; no allocation-free guarantee is made.
  Generation preserves CLDR targets verbatim. Tests verify that every generated
  target loads through the shared snapshot loader.

### Generator Inputs Are Content-Locked

- **Decision**: `internal/zone/sources.json` owns the exact version,
  commit-addressed HTTPS URL, and SHA-256 for the IANA `zone.tab` and CLDR
  `windowsZones.xml` inputs. Both generators verify their selected local file
  before writing and derive version headers and source filenames from this
  lock. Local input basenames never affect generated bytes.
- **Why**: a free-form version flag can label unrelated bytes as a trusted
  release. Content identity must be checked before generated provenance is
  emitted.
- **Rejected**: unchecked version flags, runtime downloads, vendored source
  archives, and a network/cache subsystem in the library build.
- **Contract Impact**: maintainers fetch the locked files by their recorded URL
  and pass local paths to the generators. A digest mismatch leaves existing
  generated output untouched, and the same verified bytes reproduce the same
  artifact under any local filename.

## Projection

```go
tokyo := gotime.MustLoadZone("Asia/Tokyo")
ny := gotime.MustLoadZone("America/New_York")

tokyoDT, err := instant.In(tokyo)
if err != nil {
	return err
}
nyDT, err := tokyoDT.In(ny)
if err != nil {
	return err
}
```

`In(z)` means same instant, different zone projection. It returns
`ErrOverflow` if the target zone would place the civil year outside
`0000..9999`.

## DST

`LocalDateTime.Resolve(z)` is the primitive DST projection API. It resolves a date plus clock time into a zone without hiding gaps or overlaps.

- Normal local times return `LocalResolved` with exactly one `DateTime` candidate.
- Spring-forward nonexistent local times return `LocalNonexistent` with no candidates. Calling `Only()` returns `ErrNonexistentTime`.
- Fall-back duplicate local times return `LocalAmbiguous` with all distinct chronological `DateTime` candidates. More than two candidates are possible. Calling `Only()` returns `ErrDuplicateTime`.

Resolution enumerates the offsets in the held TZif local-time types and POSIX
footer (including the default DST increment). For civil time L, it tests each
candidate L-offset using the held stdlib Location and retains only exact civil
matches. Classification happens after the entire finite set is checked; it does
not assume a minimum transition spacing or use `time.Date` normalization to
prove a gap. Nanoseconds are preserved. Candidate UTC years may lie outside the
civil `0000..9999` domain when their local projection lies within it. Stdlib owns
forward rule evaluation; no parallel transition evaluator is introduced.

`NewDateTime(d, t, z)` is the convenience path for callers that require exactly
one candidate. Formal and natural local datetime parsing with `WithZone` use the
same resolution rule. Natural parsers return civil components and never load a
zone themselves:

- Spring-forward nonexistent parse results return `StatusInvalid` and `CodeNonexistentTime`.
- Fall-back duplicate parse results return `StatusAmbiguous` with resolved candidates.

The foundation library reports ambiguity; product code decides how to resolve it.

## Language Independence

Zone and language are orthogonal. `WithZone` controls temporal projection and
supplies the calendar frame used to interpret a relative natural expression's
`WithReference` instant. `WithInputLocale(language.Tag)` controls only natural
language parsing hints. Relative natural dates/datetimes require both context
options; formal floating datetimes may omit `WithZone` and remain local.
Display language, calendar, hour cycle, and numbering system remain outside
this package.

## Forbidden

- Do not use offsets as canonical timezone identity.
- Do not call `MustLoadZone` on user input.
- Do not ignore DST gaps or duplicate local times.
- Do not put time-dependent fields in `Zone.MarshalJSON`.
- Do not expose display snapshots, formatted offsets, abbreviations, or a DST
  boolean on `Zone`.
- Do not expose timezone transition enumeration or observance types.
- Do not add `GuessZone`, `ValidateZone`, `ListZones`, or `LocalZone`; use the current API.

## Acceptance Criteria

- Zone JSON contains only `kind` and `id`, and rejects fixed-offset identities.
- Point-in-time abbreviation and numeric offset projection uses stdlib
  `time.Time.Zone`; no parallel DTO or string offset API exists.
- Public API review confirms that no transition enumeration or observance type
  exists. Synthetic TZif cases verify short-lived offsets, multiple overlaps,
  footer-only offsets, transition endpoints and snapshot consistency; the bundled
  corpus verifies that inverse resolution retains known forward projections.
- DST gaps and duplicate local times remain observable through `LocalDateTime.Resolve` and parsing with `WithZone`.
