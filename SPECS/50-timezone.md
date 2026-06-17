# Time Zones

## Overview

go-time treats IANA timezone IDs as the canonical timezone identity and uses stdlib `time.Location` for projection. Offsets are input aids; abbreviations are display metadata, not durable identity.

## Zone API

```go
var UTC Zone
var Local Zone

func LoadZone(id string) (Zone, error)
func MustLoadZone(id string) Zone
func ResolveZone(id string) (Zone, error)
func Zones() []string
func ZoneCatalogVersion() string
```

- `LoadZone` is strict IANA lookup through `time.LoadLocation`.
- `MustLoadZone` is only for source-code constants in `var` or `init` paths.
- `ResolveZone` accepts real-world zone names: exact IANA, case-insensitive IANA, Windows names, and legacy aliases handled by Go's `time.LoadLocation`.
- `ResolveZone` does not resolve timezone abbreviations. Abbreviations are point-in-time display metadata available through `Snapshot` and `Abbreviation`.
- `Zones` returns the generated IANA identifier catalog. `ZoneCatalogVersion` returns the IANA tzdb version used to generate that catalog; it does not describe the transition-rule data used by `time.LoadLocation`.
- `UTC` and `Local` mirror stdlib style; `Local` is captured from `time.Local` at init.

Use `LoadZone` for stable configuration and persisted canonical IDs. Use `ResolveZone` for CLI args, forms, migration input, and compatibility paths.

## Zone Identity

```go
zone.ID()
zone.Location()
zone.String()
zone.Equal(other)
zone.IsZero()
```

`Zone.Location()` is total: the zero zone returns `time.UTC`. `IsZero` exists only to detect whether a caller explicitly supplied a zone.

`Zone.MarshalJSON` is deterministic:

```json
{"kind":"zone","id":"Asia/Tokyo"}
```

It never emits current offset, DST state, abbreviation, or any field that would require `time.Now()`. Fixed UTC offsets are not zone identities; `ResolveZone("+08:00")` and `ResolveZone("UTC+8")` return `ErrInvalidZone`, and marshaling an internally malformed fixed-offset `Zone` returns `ErrInvalidZone` instead of emitting `{"kind":"zone","id":"+08:00"}`.

RFC 3339 values with numeric offsets parse to `Instant`. The offset is syntax for an absolute moment, not a persisted zone identity.

## Snapshot

Display-related zone data is time-dependent and must be requested explicitly:

```go
snap := zone.Snapshot(gotime.Now())
snap.ID
snap.Offset
snap.DST
snap.Abbreviation
```

JSON shape:

```json
{"id":"Asia/Tokyo","offset":"+09:00","dst":false,"abbreviation":"JST"}
```

Convenience methods `OffsetAt`, `IsDST`, and `Abbreviation` are allowed because they require an explicit `Instant`.

## Projection

```go
tokyo := gotime.MustLoadZone("Asia/Tokyo")
ny := gotime.MustLoadZone("America/New_York")

tokyoDT := instant.In(tokyo)
nyDT := tokyoDT.In(ny)
```

`In(z)` means same instant, different zone projection.

## DST

`LocalDateTime.Resolve(z)` is the primitive DST projection API. It resolves a date plus clock time into a zone without hiding gaps or overlaps.

- Normal local times return `LocalResolved` with exactly one `DateTime` candidate.
- Spring-forward nonexistent local times return `LocalNonexistent` with no candidates. Calling `Only()` returns `ErrNonexistentTime`.
- Fall-back duplicate local times return `LocalAmbiguous` with chronological `DateTime` candidates. Calling `Only()` returns `ErrDuplicateTime`.

`NewDateTime(d, t, z)` is the convenience path for callers that require exactly one candidate. Parsing a local datetime with `WithZone` uses the same resolution rule:

- Spring-forward nonexistent parse results return `StatusInvalid` and `CodeNonexistentTime`.
- Fall-back duplicate parse results return `StatusAmbiguous` with resolved candidates.

The foundation library reports ambiguity; product code decides how to resolve it.

## Language Independence

Zone and language are orthogonal. `WithZone` controls temporal projection. `WithInputLocale(language.Tag)` controls natural-language parsing hints. Display language, calendar, hour cycle, and numbering system remain outside this package.

## Forbidden

- Do not use offsets as canonical timezone identity.
- Do not call `MustLoadZone` on user input.
- Do not ignore DST gaps or duplicate local times.
- Do not put time-dependent fields in `Zone.MarshalJSON`.
- Do not add `GuessZone`, `ValidateZone`, `ListZones`, or `LocalZone`; use the current API.
