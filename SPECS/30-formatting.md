# Formatting Boundary

## One Sentence

go-time does not format. It provides semantic values and bridges to stdlib-compatible data; rendering belongs to the caller.

## Rationale

Time semantics and display semantics age at different speeds. ISO 8601, IANA zones, and Go's stdlib time model are long-lived. Locale display rules and CLDR data change frequently. Binding them together would make the foundation unstable.

The package therefore owns only stable primitives:

- `Instant`, `DateTime`, `Date`, `Time`
- `Duration`, `Period`, `Interval`
- `Zone`
- parse diagnostics

Everything user-facing goes through a bridge.

## Bridges

```go
func (i  Instant)  Std() time.Time
func (dt DateTime) Std() time.Time
func (iv Interval) StdRange() (start, end time.Time)
func (d  Duration) Std() time.Duration
```

`Date` and `Time` are unresolved civil values, so they do not have total stdlib
bridges. Combine them as `LocalDateTime`, call `Resolve(Zone)`, and bridge only a
chosen `DateTime` candidate through `DateTime.Std()`.

`Duration` also exposes clock slots:

```go
func (d Duration) Decompose() DurationComponents

type DurationComponents struct {
    Hours        int64
    Minutes      int64
    Seconds      int64
    Milliseconds int64
    Microseconds int64
    Nanoseconds  int64
}
```

`Period` exposes calendar slots directly through exported fields:

```go
p.Years
p.Months
p.Days
```

There is no `Period.Decompose()`.

## Input Locale

Parsing may accept `language.Tag` through `WithInputLocale`. This is a parsing hint only. Unicode `-u-` display extensions such as hour cycle, calendar, and numbering system are outside go-time. Controlled input phrase tables are part of parsing; they are not a display formatting layer.

## Allowed

- Return `time.Time`, `time.Duration`, `DurationComponents`, and exported `Period` fields.
- Accept `golang.org/x/text/language.Tag` as an input parsing hint.
- Let callers use stdlib `time.Format`, external formatting packages, logs, templates, or any other renderer outside this module.

## Forbidden

- No formatter types in go-time.
- No `Locale`, `HourCycle`, `Calendar`, or `Style` types.
- No methods like `dt.Format(f)` or `d.Render(locale)`.
- No CLDR, display-locale assets, or formatter data files in this repository.
- No display i18n, CLDR, message-format, display-locale, or formatter dependency.
