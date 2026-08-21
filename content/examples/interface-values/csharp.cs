// Go: type Box struct { Value any } with a Point{3, 4} inside, registered
// via gob.Register(Point{}). An interface field carries its concrete type
// along with the value, so the decoder can hand back a fully typed
// GobObject for it.
var box = Gob.Decode<GobObject>(data);
var inner = (GobObject)box["Value"]!;

string concreteType = inner.GobType;
