// A top-level non-struct value is singleton-wrapped on the wire.
byte[] data = Gob.Encode(42L);
long value = Gob.Decode<long>(data);
