// Describe a Go struct by hand when there is no C# class for it:
// the Go type name, then its fields in declaration order.
var schema = new GobSchema("Point",
    ("X", GobFieldType.Int),
    ("Y", GobFieldType.Int));

byte[] data = Gob.Encode(new Dictionary<string, object?>
{
    ["X"] = 3L,
    ["Y"] = 4L,
}, schema);
