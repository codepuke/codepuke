// Go's time.Time is a BinaryMarshaler, so it needs a codec.
// DefaultCodecs.All supplies the built-in time.Time and uuid.UUID codecs.
var when = Gob.Decode<DateTimeOffset>(data, DefaultCodecs.All);
