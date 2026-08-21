// A struct with no registered factory decodes to a GobObject, which exposes
// the wire's field names without a compile-time type.
const point = decode<GobObject>(bytes);

point.type;      // "Point"
point.has('X');  // true
point.keys();    // ["X", "Y"]
point.values();  // [3n, 4n]

const names: string[] = [];
for (const [name] of point) {
  names.push(name);
}
