byte[] data = Gob.Encode(new List<long> { 1, 2, 3 });

// Slices decode as List<object?> when no target type is given.
var items = (List<object?>)Gob.Decode(data)!;
