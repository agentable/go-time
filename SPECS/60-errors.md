# Errors

## Overview

go-time uses the same two-track model as many stdlib APIs:

- Sentinel `Err*` values for `errors.Is`.
- Typed `*TimeError` values for `errors.As` and structured metadata.

`*TimeError.Unwrap()` returns the sentinel. `Code` is for JSON, logs, and tools; it does not drive Go control flow.

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
    log.Printf("code=%s input=%q hint=%s", te.Code, te.Input, te.Hint)
}
```

Do not use `te.Code` for Go control flow when a sentinel exists.

## ParseResult Relationship

- `Parse` returns `ParseResult`, not `error`.
- `StatusInvalid` carries `ParseResult.Error`.
- `StatusAmbiguous` carries `ParseResult.Candidates`.
- Typed parsers translate non-resolved or wrong-kind results into `*TimeError`.
- Typed parsers keep ambiguity causes precise: slash-date ambiguity wraps `ErrAmbiguousDate`, while DST fall-back duplicate local times wrap `ErrDuplicateTime`.
- Warning metadata never determines sentinel identity.

Empty input in `Parse` returns `ErrEmptyInput` with `CodeEmptyInput`.

## JSON

`TimeError` JSON uses:

```json
{"code":"AMBIGUOUS_DATE","message":"...","input":"04/05/2026","hint":"..."}
```

`ParseResult` embeds errors under `error` when invalid.

## Public Panics

Only `MustLoadZone` may panic. Every other public API returns a value plus `error`, a `ParseResult`, or a deterministic value.

## Forbidden

- Do not return vague errors without actionable hints from package APIs.
- Do not encode `Parse` ambiguity as a Go `error`.
- Do not mix `Code*` and `Err*` naming.
- Do not add public panic APIs beyond `MustLoadZone`.
