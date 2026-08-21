// Go: map[string]int{"one": 1, "two": 2}
// Maps are Map instances, not plain objects — gob keys are not always strings.
const source = new Map<string, bigint>([
  ['one', 1n],
  ['two', 2n],
]);
const bytes = encode(source, { keyType: GOB_STRING, elemType: GOB_INT });

const counts = decode<Map<string, bigint>>(bytes);
counts.get('one'); // 1n

// Go's map iteration order is not deterministic, so the byte order of the
// entries varies between runs. Compare decoded values, never raw bytes.
