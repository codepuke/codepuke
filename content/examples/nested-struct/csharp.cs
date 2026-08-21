// Go: type Line struct { From, To Point }
var pointSchema = new GobSchema("Point",
    ("X", GobFieldType.Int),
    ("Y", GobFieldType.Int));

var lineSchema = new GobSchema("Line",
    ("From", GobFieldType.StructOf(pointSchema)),
    ("To", GobFieldType.StructOf(pointSchema)));

byte[] data = Gob.Encode(new Dictionary<string, object?>
{
    ["From"] = new Dictionary<string, object?> { ["X"] = 1L, ["Y"] = 2L },
    ["To"] = new Dictionary<string, object?> { ["X"] = 3L, ["Y"] = 4L },
}, lineSchema);

var line = Gob.Decode<GobObject>(data);
var to = (GobObject)line["To"]!;
long toX = (long)to["X"]!;
