// Go: type Point struct { X, Y int }
const PointSchema = new Schema('Point', { X: GOB_INT, Y: GOB_INT });

// Go's `int` is 64-bit, so integer fields are bigint on the TypeScript side.
const bytes = encode({ X: 3n, Y: 4n }, { schema: PointSchema });
