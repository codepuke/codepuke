// Go: type Status string, a named primitive. A semantic type converts
// your C# type to the underlying wire type on the way out. Here the C#
// side is an enum, so the wire value is the PascalCase name "Active"
// (a port that models Status as plain strings may write "active").
var statusType = GobFieldType.SemanticString<Status>(
    decode: s => Enum.Parse<Status>(s),
    encode: t => t.ToString(),
    zero: Status.Unknown);

var schema = new GobSchema("User",
    ("Name", GobFieldType.String),
    ("Status", statusType));

byte[] data = Gob.Encode(new Dictionary<string, object?>
{
    ["Name"] = "Ada",
    ["Status"] = Status.Active,
}, schema);
