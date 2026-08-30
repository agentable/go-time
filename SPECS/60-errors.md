# Errors

## Overview

go-time uses the same two-track model as many stdlib APIs:

- Sentinel `Err*` values for `errors.Is`.
- Typed `*TimeError` values for `errors.As` and structured metadata.

`*TimeError.Unwrap()` returns an unwrap chain containing the sentinel and, when
available, the underlying parser, loader, or codec cause. `Code` is for JSON,
logs, and tools; it does not drive Go control flow.

## ErrorCode

```go
type ErrorCode string

const (
    CodeEmptyInput        ErrorCode = "EMPTY_INPUT"
    CodeInvalidFormat     ErrorCode = "INVALID_FORMAT"
    CodeInvalidDate       ErrorCode = "INVALID_DATE"
    CodeInvalidTime       ErrorCode = "INVALID_TIME"
    CodeInvalidDuration   ErrorCode = "INVALID_DURATION"
    CodeInvalidPeriod     ErrorCode = "INVALID_PERIOD"
    CodeInvalidZone       ErrorCode = "INVALID_ZONE"
    CodeAmbiguousDate     ErrorCode = "AMBIGUOUS_DATE"
    CodeNonexistentTime   ErrorCode = "NONEXISTENT_LOCAL_TIME"
    CodeDuplicateTime     ErrorCode = "DUPLICATE_LOCAL_TIME"
    CodeIntervalReversed  ErrorCode = "INTERVAL_END_BEFORE_START"
    CodeIntervalsDisjoint ErrorCode = "INTERVALS_DISJOINT"
    CodeUnparseable       ErrorCode = "UNPARSEABLE"
    CodeOverflow          ErrorCode = "OVERFLOW"
    CodeIncompatibleTypes ErrorCode = "INCOMPATIBLE_TYPES"
)
```

## TimeError

```go
type TimeError struct {
    Code    ErrorCode `json:"code"`
    Message string    `json:"message,omitzero"`
    Input   string    `json:"input,omitzero"`
    Hint    string    `json:"hint,omitzero"`
    Err     error     `json:"-"`
}
```

Every package-generated `*TimeError` should include an actionable `Hint`.
`Message` and `Input` may contain caller-provided data. `Error()` MUST exclude
both fields and return only the fixed `ErrorCode` and matching sentinel text;
unknown codes return `time error` rather than echoing untrusted fields.
`Err` stores the unwrap chain, not necessarily the sentinel alone. Callers use
`errors.Is` for the sentinel and `errors.As` for an underlying typed cause.

## Sentinels

Each error code has a matching sentinel:

```go
var (
    ErrEmptyInput        error
    ErrInvalidFormat     error
    ErrInvalidDate       error
    ErrInvalidTime       error
    ErrInvalidDuration   error
    ErrInvalidPeriod     error
    ErrInvalidZone       error
    ErrAmbiguousDate     error
    ErrNonexistentTime   error
    ErrDuplicateTime     error
    ErrIntervalReversed  error
    ErrIntervalsDisjoint error
    ErrUnparseable       error
    ErrOverflow          error
    ErrIncompatibleTypes error
)
```

Sentinel text is a short, lowercase phrase without a `gotime:` package prefix.
This keeps it composable when callers add operation context. Sentinel identity
and `ErrorCode`, not the text, own programmatic classification.

Use names by type:

- `Code*` is machine-readable string metadata.
- `Err*` is a sentinel `error`.

## Usage

```go
if errors.Is(err, gotime.ErrAmbiguousDate) {
    // control flow
}

var te *gotime.TimeError
if errors.As(err, &te) {
    log.Printf("code=%s hint=%s", te.Code, te.Hint)
}
```

Do not use `te.Code` for Go control flow when a sentinel exists.
Apply an application-owned redaction policy before logging `te.Message` or
`te.Input`.

## ParseResult Relationship

- `Parse` returns `ParseResult`, not `error`.
- `StatusInvalid` carries `ParseResult.Error`.
- `StatusAmbiguous` carries `ParseResult.Candidates`.
- Typed parsers translate non-resolved or wrong-kind results into `*TimeError`.
- Typed parsers keep ambiguity causes precise: slash-date ambiguity wraps `ErrAmbiguousDate`, while DST fall-back duplicate local times wrap `ErrDuplicateTime`.
- Interval parsing keeps a recognized endpoint's semantic sentinel and code;
  its `TimeError.Input` is rewritten to the complete interval while the
  endpoint hint and unwrap chain remain inspectable.
- Warning metadata never determines sentinel identity.

Empty input in `Parse` returns `ErrEmptyInput` with `CodeEmptyInput`.

## JSON

`TimeError` JSON uses:

```json
{"code":"AMBIGUOUS_DATE","message":"...","input":"04/05/2026","hint":"..."}
```

`ParseResult` embeds errors under `error` when invalid.
JSON preserves `Message`, `Input`, and `Hint` for structured consumers; it is
not a sanitized log payload.

This JSON is diagnostic output only. It preserves the stable `ErrorCode` and
structured details, but `Err` and all underlying parser, loader, or codec
causes are omitted. Decoding those fields into a `TimeError` does not restore
sentinel identity or make `errors.Is` work. Go callers that need sentinel or
cause inspection must retain the original in-process error; cross-wire callers
may classify the stable `code` according to their own protocol policy.

Concrete value decoders classify wrong field types, unknown members, missing
required fields, and wrong `kind` as `ErrInvalidFormat` with a `*TimeError` and
an actionable hint. When jsonv2 supplies an underlying structural cause, it
remains reachable through `errors.As` but is excluded from `TimeError.Error()`.
Malformed top-level JSON never reaches a concrete type decoder; jsonv2 returns
its `*jsontext.SyntacticError` directly.

After structure validation, concrete value decoders map invalid dates, clock
times, exact durations, calendar periods, zones, intervals, and RFC3339 values
to their precise existing sentinel. These failures support `errors.As` to
`*TimeError`, always include a Hint, and retain an underlying parser or loader
cause when one exists.

## Public Panics

Only `MustLoadZone` may panic. Every other public API returns a value plus `error`, a `ParseResult`, or a deterministic value.

## Forbidden

- Do not return vague errors without actionable hints from package APIs.
- Do not encode `Parse` ambiguity as a Go `error`.
- Do not mix `Code*` and `Err*` naming.
- Do not add public panic APIs beyond `MustLoadZone`.
