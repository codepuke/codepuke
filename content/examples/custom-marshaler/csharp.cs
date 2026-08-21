// Go: type Celsius float64, a BinaryMarshaler that writes the temperature
// as 8 big-endian IEEE 754 bytes. CelsiusCodec implements IGobCodec<double>
// to speak that format; register it under the unqualified Go type name as
// it appears on the wire. A schema field typed Marshaler("Celsius",
// "binary") routes values through it.
var codec = new CelsiusCodec();
var schema = new GobSchema("Reading",
    ("Temp", GobFieldType.Marshaler("Celsius", "binary")));

var encoder = new GobEncoder(stream);
encoder.RegisterCodec("Celsius", codec);
encoder.Encode(new Dictionary<string, object?> { ["Temp"] = 21.5 }, schema);

stream.Position = 0;

var decoder = new GobDecoder(stream);
decoder.RegisterCodec("Celsius", codec);
var reading = decoder.Decode<GobObject>();
double celsius = (double)reading["Temp"]!;
