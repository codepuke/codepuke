# Go: type Celsius float64, with MarshalBinary / UnmarshalBinary writing
# eight big-endian IEEE-754 bytes.
def encode_celsius(value: float) -> bytes:
    return struct.pack(">d", value)

def decode_celsius(data: bytes) -> float:
    return struct.unpack(">d", data)[0]

# Register under the unqualified Go type name, on both sides.
buf = io.BytesIO()
enc = Encoder(buf)
enc.register_codec("Celsius", encode_celsius, marshaler_type="binary")
enc.encode_gob_encoded(21.5, "Celsius")

buf.seek(0)
dec = Decoder(buf)
dec.register_codec("Celsius", decode_celsius)
temperature = dec.decode()   # 21.5
