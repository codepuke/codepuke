// Go: 42 — top-level scalars carry their type on the wire, so no schema.
const bytes = encode(42n);
const answer = decode<bigint>(bytes); // 42n

// The other scalar kinds work the same way.
decode<string>(encode('gob')); // "gob"
decode<boolean>(encode(true)); // true
decode<number>(encode(2.5)); // 2.5 (a Go float64)
