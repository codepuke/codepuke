// Go: type Line struct { From, To Point }
const PointSchema = new Schema('Point', { X: GOB_INT, Y: GOB_INT });

// A Schema is itself a field type, so it nests directly.
const LineSchema = new Schema('Line', {
  From: PointSchema,
  To: PointSchema,
});

const bytes = encode(
  { From: { X: 1n, Y: 2n }, To: { X: 3n, Y: 4n } },
  { schema: LineSchema },
);

const line = decode<GobObject>(bytes);
const to = line.get('To') as GobObject;
to.get('X'); // 3n
