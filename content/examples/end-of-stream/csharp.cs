// TryDecode returns false at end of stream; Decode throws
// EndOfStreamException instead. End of stream is never swallowed silently.
var decoder = new GobDecoder(stream);
var values = new List<object?>();
while (decoder.TryDecode(out object? value))
    values.Add(value);
