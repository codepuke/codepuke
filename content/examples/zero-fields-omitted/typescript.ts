const full = encode({ X: 3n, Y: 4n }, { schema: PointSchema });
const partial = encode({ X: 3n, Y: 0n }, { schema: PointSchema });

// Y is zero in the second value, so it is omitted from the wire entirely.
partial.length < full.length; // true

// The decoder restores every omitted field to its zero value.
const point = decode<GobObject>(partial);
point.get('X'); // 3n
point.get('Y'); // 0n
