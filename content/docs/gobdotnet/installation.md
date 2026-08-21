
# Installation

gobdotnet is not yet published to NuGet. Add it to your solution as a project
reference:

```xml
<ItemGroup>
  <ProjectReference Include="../gobdotnet/GobDotNet/GobDotNet.csproj" />
  <!-- Optional: source generator for AOT support and compile-time validation -->
  <ProjectReference Include="../gobdotnet/GobDotNet.SourceGenerators/GobDotNet.SourceGenerators.csproj"
                    OutputItemType="Analyzer" ReferenceOutputAssembly="false" />
</ItemGroup>
```

The second reference is optional but recommended. Without it, `[GobStruct]` classes
still work through a reflection fallback; with it, schemas are computed at compile
time and the library is NativeAOT-compatible. See [Source generator](source-generator).

The library itself takes **no external runtime dependencies** — BCL only.

## Requirements

- .NET 10 or later.
- Go on `PATH` only if you want to run the cross-validation test suite; it is not
  needed to use the library.

## First encode and decode

```csharp
using GobDotNet;

// Decode a Go-produced gob stream.
byte[] gobBytes = File.ReadAllBytes("data.gob");
var point = Gob.Decode<GobObject>(gobBytes);
Console.WriteLine($"X={point["X"]}, Y={point["Y"]}");

// Encode a struct to send to a Go service.
byte[] encoded = Gob.Encode(new Point { X = 1, Y = 2 });

// Round-trip a payload containing time.Time and uuid.UUID.
var evt = Gob.Decode<GobObject>(File.ReadAllBytes("event.gob"), DefaultCodecs.All);
// evt["CreatedAt"] is a DateTimeOffset; evt["Id"] is a Guid
```

`Point` here is a `[GobStruct]` class — see [Schemas](schemas) for how to declare one.

## Convenience API

`Gob.Encode` and `Gob.Decode` create a fresh encoder or decoder per call, so they are
inherently thread-safe. Reach for [`GobEncoder`](encoding) / [`GobDecoder`](decoding)
directly when you need to write or read more than one value over a single stream.

```csharp
byte[] Gob.Encode<T>(T value, IReadOnlyDictionary<string, IGobCodec>? codecs = null);
byte[] Gob.Encode(IDictionary<string, object?> value, GobSchema schema,
                  IReadOnlyDictionary<string, IGobCodec>? codecs = null);
object? Gob.Decode(byte[] data, IReadOnlyDictionary<string, IGobCodec>? codecs = null);
T Gob.Decode<T>(byte[] data, IReadOnlyDictionary<string, IGobCodec>? codecs = null);
```
