byte[] data = Gob.Encode(new Dictionary<string, long> { ["one"] = 1, ["two"] = 2 });

// Maps decode as Dictionary<object, object?> when no target type is given.
var m = (Dictionary<object, object?>)Gob.Decode(data)!;
