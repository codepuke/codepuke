var schema = new GobSchema("Point",
    ("X", GobFieldType.Int),
    ("Y", GobFieldType.Int));

byte[] full = Gob.Encode(new Dictionary<string, object?>
{
    ["X"] = 3L,
    ["Y"] = 4L,
}, schema);

// Y is zero, so gob omits it from the wire entirely...
byte[] partial = Gob.Encode(new Dictionary<string, object?>
{
    ["X"] = 3L,
    ["Y"] = 0L,
}, schema);

// ...and the decoder puts the zero value back on the way out.
var decoded = Gob.Decode<GobObject>(partial);
long y = (long)decoded["Y"]!;
