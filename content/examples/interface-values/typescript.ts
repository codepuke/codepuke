// Go: type Box struct { Value any }
const PointSchema = new Schema('Point', { X: GOB_INT, Y: GOB_INT });
const BoxSchema = new Schema('Box', { Value: GOB_INTERFACE });

// An interface value travels with its concrete type name, which Go qualifies
// by package — "main.Point", not "Point". That is the name to register, and
// it must match whatever the Go side passed to gob.Register.
const enc = new GobEncoder();
enc.register('main.Point', PointSchema);
enc.encode(
  { Value: new GobObject('main.Point', { X: 3n, Y: 4n }) },
  { schema: BoxSchema },
);

// The concrete value comes back as a GobObject carrying its own field names.
const dec = new GobDecoder(enc.bytes());
const box = dec.decode<GobObject>();
const point = box.get('Value') as GobObject;

point.type;      // "Point"
point.get('X');  // 3n
