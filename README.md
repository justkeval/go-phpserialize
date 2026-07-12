# go-phpserialize

A Go library for serializing and deserializing the PHP `serialize()` / `unserialize()`
data format. The public API mirrors the standard library's `encoding/json`.

```go
import "github.com/justkeval/go-phpserialize"
```

## Goals

- Familiar API and Developer Experience as `encoding/json` with `Marshal()`/`Unmarshal()`.
- Decoding semantics and flexibility of `encoding/json` (e.g. struct tags, zero value for missing fields, unknown fields ignored). This was the main motivation for creating this lib as I lacked this from other libs.
- Be as close to performance as `encoding/json`. The `benchmark_test.go` file contains simple benchmarking code for comparison against encoding json, we can add more variation in the benchmark itself also.
- Correctness and edge case handling. Need to handle every case of semantic differences between types of php and go.

## Design

The package is split into four independent layers that communicate only through a
single intermediate representation, [`Value`](value.go):

```
Go value ──Encode──▶ Value ──Write──▶ PHP bytes
PHP bytes ──Parse──▶ Value ──Decode─▶ Go value
```

- **Parser** (`parser.go`) turns PHP serialized bytes into a `Value`. It is a strict
  recursive-descent parser and never inspects any Go destination type.
- **Writer** (`writer.go`) turns a `Value` into canonical PHP serialized bytes.
- **Encode** (`encode.go`) turns a Go value into a `Value` using reflection.
- **Decode** (`decode.go`) turns a `Value` into a Go value using reflection.

Reflection lives only in the encode/decode layers; the parser and writer operate purely
on bytes and `Value`s.

## API

```go
func Marshal(v any) ([]byte, error)
func Unmarshal(data []byte, v any) error

func Parse(data []byte) (Value, error)
func Write(v Value) ([]byte, error)

func Encode(v any) (Value, error)
func Decode(v Value, dst any) error
```

`Marshal` is `Encode` followed by `Write`; `Unmarshal` is `Parse` followed by `Decode`.
Use `Parse`/`Write` directly when you need access to the intermediate representation —
for example to read a PHP object's class name, which `Unmarshal` discards.

## Supported PHP types

`N` (null), `b` (bool), `i` (int), `d` (double), `s` (string), `a` (array) and
`O` (object). Custom serialized objects (`C`) and references (`R`/`r`) are intentionally
not supported; encountering them returns `ErrUnsupportedType`.

## Type mapping

### Go → PHP (encode)

| Go | PHP |
| --- | --- |
| `nil`, nil pointer/interface | null |
| `bool` | bool |
| all int/uint types | int |
| `float32`/`float64` | double |
| `string` | string |
| `[]byte` | string (raw bytes) |
| slice, array | sequential array (keys `0..n-1`) |
| map | array (keys sorted for determinism) |
| struct | associative array, or object if it implements `PHPClass` |
| pointer, interface | the value they point to |

A struct becomes a PHP object when its type implements:

```go
type PHPClass interface {
    PHPClassName() string
}
```

### PHP → Go (decode into `any`)

| PHP | Go |
| --- | --- |
| null | `nil` |
| bool | `bool` |
| int | `int64` |
| double | `float64` |
| string | `string` |
| sequential array (keys `0..n-1`) | `[]any` |
| associative/mixed array | `map[any]any` |
| object | `map[string]any` (class name discarded) |

Decoding into concrete types works as expected: arrays into slices/arrays/maps/structs,
objects into structs/maps, with safe numeric conversions and overflow checking.

## Struct tags

Struct tags follow `encoding/json` conventions under the `php` key:

```go
type T struct {
    Name  string `php:"name"`
    Email string `php:"email,omitempty"`
    Internal int `php:"-"`
}
```

## Float formatting

Doubles are formatted exactly as PHP's `serialize()` does with the default
`serialize_precision = -1`: the shortest decimal string that round-trips, laid out by
PHP's `php_gcvt` algorithm (including `INF`, `-INF` and `NAN`).
