const PointSchema = new Schema('Point', { X: GOB_INT, Y: GOB_INT });

// Reusing one encoder keeps the type state: the Point type definition is
// written once, before the first value, and never repeated.
const enc = new GobEncoder();
enc.encode({ X: 3n, Y: 4n }, { schema: PointSchema });
enc.encode({ X: 5n, Y: 6n }, { schema: PointSchema });

// bytes() drains the accumulated buffer but keeps the type state.
const stream = enc.bytes();

const dec = new GobDecoder(stream);
const points: GobObject[] = [];
for (const value of dec) {
  points.push(value as GobObject);
}
// points holds Point{3, 4} and Point{5, 6}
