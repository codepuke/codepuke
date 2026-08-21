
# Codecs

Go types that implement `GobEncoder`, `encoding.BinaryMarshaler`, or
`encoding.TextMarshaler` do not go on the wire as structs — they go as an opaque byte
blob tagged with the type's **unqualified** name (`"Time"`, `"UUID"`). A codec is what
turns that blob into a C# value.

Without one, the value decodes to a `GobEncoded`:

```csharp
var raw = Gob.Decode<GobObject>(bytes);
var enc = (GobEncoded)raw["CreatedAt"]!;   // enc.TypeName == "Time", enc.Data is byte[15]
```

## Built-in codecs

`DefaultCodecs.All` supplies codecs for the two types you almost always want.

:::examples time-values

:::examples uuid-values

**`TimeCodec`** (`DefaultCodecs.All["Time"]`) maps Go's `time.Time` to
`DateTimeOffset`, in both directions:

- UTC times decode with a `TimeSpan.Zero` offset.
- `DateTimeOffset` has 100 ns tick precision, so sub-tick nanoseconds from Go are
  truncated on decode.
- The wire format stores whole-minute offsets only; sub-minute offsets are truncated
  on encode. Real-world time zones are always whole minutes, so this is safe in
  practice.
- Go's IANA zone name (`"America/New_York"`) cannot be represented in a
  `DateTimeOffset` — only the numeric offset survives a round-trip.

**`GuidCodec`** (`DefaultCodecs.All["UUID"]`) maps a 16-byte RFC 4122 big-endian UUID
to `Guid`. It is compatible with `github.com/google/uuid`, `github.com/gofrs/uuid`,
and `github.com/satori/go.uuid` without per-package configuration: all three produce
the same wire bytes, and gob transmits only the unqualified name `"UUID"`.

## Writing your own codec

Implement `IGobCodec<T>` and register it under the Go type's unqualified wire name:

:::examples custom-marshaler

`MarshalerType` returns `"gob"`, `"binary"`, or `"text"` — matching which Go marshaler
interface the type implements.

### Two rules worth knowing

**Register post-construction, not via the constructor.** A codec passed through the
`codecs` constructor parameter is only consulted if it implements an interface
internal to the library. Your own codec implements the public `IGobCodec<T>` only, so
a constructor-passed codec is ignored and the value comes back as `GobEncoded`. Use
`RegisterCodec<T>()` after construction, as the example above does, and both halves
of your codec are used: `Decode` on the way in, `Encode` on the way out.

**Encoding needs the field described as a marshaler type.** The encoder dispatches on
the C# type, and a plain `double` encodes as an ordinary gob float. To route a value
through your codec, name the Go type in the schema with
`GobFieldType.Marshaler("Celsius", "binary")`, as the example above does. The second
argument matches the codec's `MarshalerType`.
