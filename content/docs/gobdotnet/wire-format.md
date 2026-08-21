
# Wire format notes

gobdotnet reproduces Go's `encoding/gob` behaviour, quirks included. Where the Go
source does something surprising, the library does the same thing — that fidelity is
the whole point. These are the details that most often surprise people.

## Compatibility

- **User type ids start at 65.** Go's constant is `firstUserId = 65`, and gobdotnet
  matches it. But Go's in-process type registry accumulates ids across encode calls,
  so a fresh C# encoder and a long-lived Go encoder assign *different* ids to the same
  struct even though the decoded values are identical. Validate struct output by
  decoding and comparing values structurally, not byte-for-byte. Byte-level comparison
  is only reliable for scalars, which carry no user type ids.
- **Map iteration order is non-deterministic in Go.** Never byte-compare gob output
  that contains a map.
- **Field order must match Go's source declaration order.** Use
  `[GobField(Order = N)]` when your C# property order differs.
- **Zero-valued fields are omitted entirely.** The decoder pre-populates every field
  with its zero value before running the delta loop.
- **The first field delta is 1, not 0.** Field indices start at -1, and a delta of 0
  is the struct terminator.
- **Nested struct fields are unwrapped** — no type definition, no byte-count prefix,
  just raw delta-encoded bytes.
- **Collection wire types carry an empty `CommonType.Name`,** so the `Id` field arrives
  with delta=2, skipping the absent name.
- **Top-level non-struct values use a singleton wrapper:** the payload is
  `0x00` followed by the encoded value.
- **Float bytes are reversed.** Go encodes floats as byte-reversed IEEE 754, then as an
  unsigned int, which compresses trailing zeros well.
- **Marshaler type names are unqualified** on the wire (`"Time"`, `"UUID"`) because Go
  derives them from `reflect.Type.Name()`. Interface concrete-type registration is the
  opposite — it uses the qualified name (`"main.Point"`).

## Limitations

- **No async API.** `EncodeAsync` / `DecodeAsync` are explicitly out of scope. Use a
  thread-pool worker with the synchronous API if you need one.
- **No pointer types.** Go pointers are transparent in gob; gobdotnet does not model
  them.
- **No `BigInteger`.** Throws `GobEncodeException` at schema derivation — fail loud,
  never silently truncate.
- **Array length is not preserved on decode.** Go `[3]int` decodes to `object?[3]`
  and the fixed-length annotation is lost. Re-encoding with `GobFieldType.ArrayOf`
  restores wire fidelity.
- **`time.Duration` precision.** `TimeSpan` has 100 ns tick precision; sub-tick
  nanoseconds are truncated on decode.
- **`time.Time` offset narrowing.** Sub-minute offsets are truncated on encode.
- **Zone name loss.** Go's IANA zone name does not survive a round-trip through
  `DateTimeOffset`.
- **Semantic types are encoder-side only.** The decoder returns the underlying wire
  primitive; converting back is the caller's job. See [Schemas](schemas).
- **Custom codecs are decode-side, and must be registered post-construction.** See
  [Codecs](codecs).

## Performance

Measured on an Apple M3 Max, .NET 10.0.5, BenchmarkDotNet short job, against
Newtonsoft.Json for equivalent payloads. The target is *within 2× of
Newtonsoft.Json* — gobdotnet does not try to compete with MessagePack-CSharp.

| Scenario | Gob | JSON | Ratio |
|---|---|---|---|
| Scalar int encode | 161 ns | 86 ns | 1.9× |
| Scalar int decode | 210 ns | 123 ns | 1.7× |
| Scalar string encode | 183 ns | 124 ns | 1.5× |
| Scalar string decode | 242 ns | 178 ns | 1.4× |
| Struct (2 fields) encode | 439 ns | 139 ns | 3.2× |
| Struct (2 fields) decode | 555 ns | 336 ns | 1.7× |
| Nested struct encode | 1,058 ns | 309 ns | 3.4× |
| Nested struct decode | 1,649 ns | 875 ns | 1.9× |
| Slice of 1000 encode | 15,489 ns | 17,119 ns | **0.9× (gob faster)** |
| Slice of 1000 decode | 10,001 ns | 29,013 ns | **0.3× (gob faster)** |
| Map of 1000 encode | 35,430 ns | 31,307 ns | 1.1× |
| Map of 1000 decode | 55,533 ns | 77,605 ns | **0.7× (gob faster)** |
| Mixed round-trip | 1,382 ns | 876 ns | 1.6× |

Scalars and mixed payloads sit inside the target. Small-struct encode is 3–4× slower
than JSON (dictionary lookup overhead in the benchmark setup). Collections are
consistently *faster* than JSON — the binary format is more compact and skips text
parsing entirely.
