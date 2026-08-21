// Works with google/uuid, gofrs/uuid, and satori/go.uuid alike — all three
// marshal to the same 16-byte RFC 4122 big-endian form.
var id = Gob.Decode<Guid>(data, DefaultCodecs.All);
