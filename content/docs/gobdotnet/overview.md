
# gobdotnet

A pure C# encoder and decoder for Go's [`encoding/gob`](https://pkg.go.dev/encoding/gob)
binary serialization format. Sister library to pygob (Python) and gobts (TypeScript).

Any byte stream produced by Go's encoder decodes correctly in C#. Any byte stream
produced by gobdotnet decodes correctly in Go. Wire fidelity is the top priority:
where Go's implementation has a quirk, gobdotnet reproduces it rather than "fixing" it.

**Targets:** .NET 10, no external runtime dependencies, NativeAOT-compatible via the
source generator.

## At a glance

Encoding a struct:

:::examples encode-struct

Decoding one back:

:::examples decode-struct

## Core concepts

**`GobObject`** is what you get when decoding a Go struct with no registered C# type.
It behaves like a read-only dictionary with string keys, and carries the Go type name
(`GobType`) and the schema needed to re-encode it.

**`GobSchema`** describes a struct type: its Go name and an ordered list of
`(string Name, GobFieldType Type)` field descriptors. Derive one automatically from a
`[GobStruct]` class, or build it by hand — see [Schemas](schemas).

**`GobFieldType`** is the type descriptor for a single field: `GobFieldType.Int`,
`GobFieldType.String`, `GobFieldType.SliceOf(...)`, and so on. These correspond
directly to Go's wire type ids.

**`GobEncoded`** wraps the raw bytes of a Go type implementing `GobEncoder`,
`BinaryMarshaler`, or `TextMarshaler` when no C# codec is registered for it. Inspect
`.TypeName` and `.Data`, or register a codec to get a typed value instead — see
[Codecs](codecs).

## Type mapping

| Go type | C# type | Notes |
|---|---|---|
| `int`, `int64` | `long` | Go's default int size |
| `uint`, `uint64` | `ulong` | |
| `int32`, `int16`, `int8` | `int`, `short`, `sbyte` | Encoded as signed gob int |
| `uint32`, `uint16`, `uint8` | `uint`, `ushort`, `byte` | Encoded as unsigned gob int |
| `bool` | `bool` | |
| `float64` | `double` | |
| `float32` | `float` | |
| `complex128` | `System.Numerics.Complex` | |
| `string` | `string` | |
| `[]byte` | `byte[]` | |
| `[]T` | `List<T>` or `T[]` | Decodes as `List<object?>`; any `IEnumerable<T>` encodes |
| `[N]T` | fixed-length array via `GobFieldType.ArrayOf` | Length is not preserved on decode |
| `map[K]V` | `Dictionary<K, V>` | Decodes as `Dictionary<object, object?>` |
| `struct` | `[GobStruct]` POCO or `GobObject` | POCO requires registration |
| `interface{}` | `object?` | Concrete value embedded; structs become `GobObject` |
| `time.Time` | `DateTimeOffset` | Requires `DefaultCodecs.All` |
| `uuid.UUID` | `Guid` | Requires `DefaultCodecs.All` |
| `time.Duration` | `TimeSpan` via `GobFieldType.Duration` | 100 ns tick precision |
| `GobEncoder` / `BinaryMarshaler` / `TextMarshaler` | `GobEncoded` or a custom type | Register a codec for typed decoding |

## Where to go next

- [Installation](installation) — add the library to your solution.
- [Schemas](schemas) — describe your Go types to gobdotnet.
- [Encoding](encoding) and [Decoding](decoding) — the two halves of the API.
- [Codecs](codecs) — `time.Time`, `uuid.UUID`, and your own marshaler types.
- [Source generator](source-generator) — compile-time schemas and AOT support.
- [Wire format notes](wire-format) — compatibility details and limitations.
