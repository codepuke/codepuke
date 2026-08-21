
# Encoding

`GobEncoder` is a stream-oriented, thread-safe encoder. It keeps type-definition state
across `Encode` calls: the same schema reuses the same type id, exactly as Go's wire
protocol requires.

```csharp
var enc = new GobEncoder(stream);                        // no codecs
var enc = new GobEncoder(stream, DefaultCodecs.All);     // with time/UUID codecs

enc.Encode(new Point { X = 1, Y = 2 });   // [GobStruct] POCO
enc.Encode(dict, schema);                 // explicit schema
enc.Encode(gobObject);                    // re-encode a decoded GobObject

enc.Register("main.Point", schema);       // concrete type for an interface{} field
enc.RegisterCodec("MyType", myCodec);     // custom marshaler codec
```

**The caller owns the stream.** `GobEncoder` does not implement `IDisposable` and
never closes what you hand it.

## Scalars

A top-level value that is not a struct is singleton-wrapped on the wire — the payload
is `0x00` followed by the encoded value.

:::examples encode-scalars

## Collections

:::examples encode-slice

:::examples encode-map

> Go's map iteration order is non-deterministic, so gob output containing a map is not
> byte-stable. Compare decoded values structurally, never byte-for-byte.

## Nested structs

A nested struct field is *unwrapped* on the wire: no type definition, no byte-count
prefix, just raw delta-encoded bytes followed by the `0x00` terminator.

:::examples nested-struct

## Zero values are omitted

Gob does not transmit zero-valued fields at all. The decoder pre-populates every field
with its zero value before reading the delta stream, so what comes out matches what
went in.

:::examples zero-fields-omitted

## Multiple values over one stream

Reuse a single encoder to write many messages. The type definition for `Point` goes
out once, ahead of the first value; later messages reference it by id.

:::examples stream-multiple-values

## Interface fields

Before encoding a struct with an `interface{}` field, register the concrete type's
**qualified** Go name and schema:

```csharp
var pointSchema = new GobSchema("Point", ("X", GobFieldType.Int), ("Y", GobFieldType.Int));
var containerSchema = new GobSchema("Container",
    ("Name", GobFieldType.String),
    ("Value", GobFieldType.Interface));

var enc = new GobEncoder(stream);
enc.Register("main.Point", pointSchema);   // qualified Go name

var point = new GobObject("main.Point", pointSchema,
[
    new KeyValuePair<string, object?>("X", 7L),
    new KeyValuePair<string, object?>("Y", 13L)
]);
enc.Encode(new Dictionary<string, object?> { ["Name"] = "hello", ["Value"] = point },
    containerSchema);
```

Types annotated with `[GobStruct]` are auto-registered by the source generator — no
manual `Register()` call is needed for those.

> Interface concrete-type registration uses the **qualified** Go name (`"main.Point"`).
> This is distinct from marshaler type names, which are unqualified on the wire
> (`"Time"`, `"UUID"`) because Go derives them from `reflect.Type.Name()`.

## Errors

```csharp
try { enc.Encode(value); }
catch (GobEncodeException ex) { /* unsupported type, missing codec, schema error */ }
```

`BigInteger` as a `[GobStruct]` property type throws `GobEncodeException` at schema
derivation. That is deliberate: fail loud rather than silently truncate.

## Thread safety

`GobEncoder` serializes concurrent calls through an internal lock — `Encode`,
`Register`, and `RegisterCodec` all share it per instance.

```csharp
var enc = new GobEncoder(sharedStream);
Parallel.ForEach(items, item => enc.Encode(item));   // safe
```

`Gob.Encode` is inherently thread-safe: every call uses a fresh encoder.
