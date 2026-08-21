// One encoder over one stream: the Point type definition is sent once,
// then every later message references it by id.
var enc = new GobEncoder(stream);
enc.Encode(new Point { X = 3, Y = 4 });
enc.Encode(new Point { X = 5, Y = 6 });

stream.Position = 0;

var dec = new GobDecoder(stream);
var first = dec.Decode<GobObject>();
var second = dec.Decode<GobObject>();
