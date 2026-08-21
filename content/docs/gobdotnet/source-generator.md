
# Source generator

`GobDotNet.SourceGenerators` is an incremental Roslyn generator that runs at compile
time on every `partial` class decorated with `[GobStruct]`. It is the primary path for
schema derivation; reflection is the documented fallback.

:::examples source-generator

## What it generates

For each eligible class:

- A cached `GobSchema` static field, derived from **property declaration order**. This
  is the reliable ordering — reflection has to infer order from `MetadataToken`, which
  happens to match source order on current .NET runtimes but is not ECMA-guaranteed.
- An `IGobStructGenerated` implementation with `CreateFromFields` and `WriteFields`:
  plain code, no runtime reflection, fully NativeAOT-compatible.

`GobSchema.For<T>()` checks for `IGobStructGenerated` first and uses the generated
schema, falling back to reflection only for non-`partial` classes.

Types annotated with `[GobStruct]` are also auto-registered as interface concrete
types, so encoding a struct with an `interface{}` field needs no manual
`Register()` call for them.

## Enabling it

Reference the generator project as an analyzer — see [Installation](installation):

```xml
<ProjectReference Include="../gobdotnet/GobDotNet.SourceGenerators/GobDotNet.SourceGenerators.csproj"
                  OutputItemType="Analyzer" ReferenceOutputAssembly="false" />
```

If the class is nested inside another type, every containing type must also be
declared `partial`.

## Compile-time diagnostics

| Code | Condition |
|---|---|
| `GOB001` | `[GobStruct]` class is not `partial` — the generator cannot extend it |
| `GOB002` | Mixed `[GobField(Order = N)]` usage — some properties set `Order`, others do not |
| `GOB003` | Unsupported property type (for example `BigInteger`) |
| `GOB004` | `[GobStruct]` on an abstract class or an interface |

`GOB001` is informational, not an error: a non-`partial` class silently falls back to
reflection and keeps working. The generator and reflection paths are held to
behavioural equivalence by the test suite.

`GOB003` is deliberately fatal. `BigInteger` has no gob wire representation, and the
library fails loud rather than truncating your data.
