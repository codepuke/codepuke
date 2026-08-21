
# Schemas

Gob is a self-describing format, so decoding a stream needs no advance knowledge of
its types. Encoding does: gobdotnet has to know the Go type name and the field order
to put on the wire. That description is a `GobSchema`, and there are two ways to get
one.

## The `[GobStruct]` attribute

The primary way. Add `partial` to enable the source generator, which computes the
schema at compile time from property declaration order.

:::examples source-generator

Per-property overrides live on `[GobField]`:

```csharp
[GobStruct("Person")]
public partial class Person
{
    [GobField(Order = 1)]             // explicit wire order when C# order differs from Go
    public string Name { get; set; } = "";

    [GobField(Order = 2)]
    public long Age { get; set; }

    [GobField(Name = "home_city")]    // override the field name on the wire
    public string HomeCity { get; set; } = "";

    [GobField(Ignore = true)]         // skip this property entirely
    public string CacheKey { get; set; } = "";
}
```

Field ordering on the wire **must** match the Go struct's source declaration order.
Use `[GobField(Order = N)]` when your C# property order differs from Go's. If any
property sets `Order`, all of them must — otherwise the generator reports `GOB002`.

## An explicit `GobSchema`

For plain dictionaries, `GobObject` re-encoding, or any case where a POCO is not
practical, describe the struct directly:

:::examples define-schema

You can also derive a schema from a `[GobStruct]` type at run time:

```csharp
GobSchema schema = GobSchema.For<Point>();          // generator path, reflection fallback
GobSchema schema = GobSchema.For(typeof(Point));
```

## `GobFieldType` descriptors

```csharp
// Primitives
GobFieldType.Bool
GobFieldType.Int        // signed: long, int, short, sbyte
GobFieldType.UInt       // unsigned: ulong, uint, ushort, byte
GobFieldType.Float      // double, float
GobFieldType.Bytes      // byte[]
GobFieldType.String
GobFieldType.Complex    // System.Numerics.Complex
GobFieldType.Interface  // object?

// Well-known semantic types
GobFieldType.Duration   // TimeSpan ↔ int64 nanoseconds

// Composites
GobFieldType.SliceOf(GobFieldType.Int)
GobFieldType.MapOf(GobFieldType.String, GobFieldType.Int)
GobFieldType.ArrayOf(GobFieldType.Int, length: 3)   // Go fixed-length array
GobFieldType.StructOf(nestedSchema)

// Marshaler types
GobFieldType.Marshaler("Time", "gob")     // Go time.Time (implements GobEncoder)
GobFieldType.Marshaler("UUID", "binary")  // uuid.UUID (implements BinaryMarshaler)
```

`GobSchema`, `GobFieldType`, `GobObject`, and `GobEncoded` are all immutable and safe
to share across threads without synchronization.

## Semantic types

Go programs often use named primitives — `type Status string`, `type Count int64`.
A semantic type converts your C# type to the underlying wire primitive on the way out:

:::examples semantic-type

Available factories: `GobFieldType.SemanticInt<T>`, `SemanticUInt<T>`,
`SemanticString<T>`, and `SemanticFloat<T>`.

> Semantic conversion is **encoder-side only**. The decoder returns the underlying
> wire primitive (`long`, `ulong`, `double`, or `string`); converting back to your
> semantic type is the caller's responsibility.

`GobFieldType.Duration` is a built-in semantic type over `int64` nanoseconds:

```csharp
var schema = new GobSchema("Event",
    ("Name", GobFieldType.String),
    ("Duration", GobFieldType.Duration));   // TimeSpan ↔ int64 nanoseconds

var encoded = Gob.Encode(
    new Dictionary<string, object?> { ["Name"] = "ping", ["Duration"] = TimeSpan.FromSeconds(5) },
    schema);

// Decoding without a Duration schema yields the raw wire value:
var decoded = Gob.Decode<GobObject>(encoded);
// decoded["Duration"] == 5_000_000_000L
```
