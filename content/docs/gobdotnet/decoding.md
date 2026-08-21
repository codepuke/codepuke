
# Decoding

`GobDecoder` is a stream-oriented, thread-safe decoder. It maintains a type registry
across calls, so type definitions received in earlier messages are reused by later
ones.

```csharp
var dec = new GobDecoder(stream);
var dec = new GobDecoder(stream, DefaultCodecs.All);

object? value = dec.Decode();               // GobObject, long, List<object?>, …
Point point   = dec.Decode<Point>();        // casts; InvalidCastException on mismatch
bool ok       = dec.TryDecode(out var v);   // false at end of stream
bool ok       = dec.TryDecode<Point>(out var p);

dec.Register<Point>("Point");               // map a Go struct name → C# type
dec.RegisterCodec("Time", TimeCodec.Instance);
```

**The caller owns the stream.** `GobDecoder` does not implement `IDisposable`.

## Structs become `GobObject`

With no registered C# type, a Go struct decodes to a `GobObject` — a read-only
dictionary that also carries the Go type name and the schema needed to re-encode it.

:::examples decode-struct

Because a `GobObject` knows its own schema, you can feed it straight back to an
encoder: `enc.Encode(gobObject)` reproduces an equivalent stream.

## End of stream

`Decode()` throws `EndOfStreamException` when the stream is exhausted; `TryDecode`
returns `false` instead. End of stream is never swallowed silently.

:::examples end-of-stream

## Interface values

No registration is required on the decode side. The stream is self-describing: the
concrete type definition is embedded inline and the decoder reconstructs the value
automatically. Interface fields decode as `GobObject` with `GobType` set to the
unqualified Go type name (`"Point"`); the qualified registration name
(`"main.Point"`) travels on the wire but only the embedded type definition names
the decoded object.

:::examples interface-values

## Errors

```csharp
// End of stream
try { dec.Decode(); }
catch (EndOfStreamException) { /* stream exhausted */ }

// …or avoid the exception entirely:
if (!dec.TryDecode(out var value)) { /* end of stream */ }

// Malformed wire data
try { dec.Decode(); }
catch (GobDecodeException ex) { }

// Type mismatch
try { dec.Decode<Point>(); }
catch (InvalidCastException) { /* decoded value is not a Point */ }
```

Exception hierarchy:

```
GobException : Exception
├── GobDecodeException   (malformed wire data)
└── GobEncodeException   (unsupported type, missing codec, schema error)
```

`EndOfStreamException` is the BCL type, not a gob-specific one.

## Thread safety

`GobDecoder` serializes concurrent calls through an internal lock — `Decode`,
`TryDecode`, `Register`, and `RegisterCodec` all share it per instance.

```csharp
var dec = new GobDecoder(sharedStream);
// Each Decode() returns the *next* value; concurrent callers get different
// values, not the same value duplicated.
```

`Gob.Decode` is inherently thread-safe: every call uses a fresh decoder.
