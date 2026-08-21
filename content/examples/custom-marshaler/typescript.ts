// Go: type Celsius float64, with MarshalBinary / UnmarshalBinary writing
// eight big-endian IEEE-754 bytes.
const CelsiusCodec: Codec<number> = {
  kind: 'binary',
  encode(value) {
    const out = new Uint8Array(8);
    new DataView(out.buffer).setFloat64(0, value, false);
    return out;
  },
  decode(bytes) {
    const view = new DataView(bytes.buffer, bytes.byteOffset, bytes.byteLength);
    return view.getFloat64(0, false);
  },
};

// Register under the unqualified Go type name, on both sides.
const enc = new GobEncoder();
enc.registerCodec('Celsius', CelsiusCodec);
enc.encode(21.5, { marshalerType: 'Celsius', marshalerKind: 'binary' });

const dec = new GobDecoder(enc.bytes());
dec.registerCodec('Celsius', CelsiusCodec);
const temperature = dec.decode<number>(); // 21.5
