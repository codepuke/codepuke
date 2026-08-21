# uuid.UUID is a BinaryMarshaler, wire type name "UUID".
buf = io.BytesIO()
enc = Encoder(buf, codecs=DEFAULT_CODECS)
enc.encode_gob_encoded(uuid.UUID("6ba7b810-9dad-11d1-80b4-00c04fd430c8"), "UUID")

ident = pygob.decode(buf.getvalue(), codecs=DEFAULT_CODECS)
print(ident)   # 6ba7b810-9dad-11d1-80b4-00c04fd430c8
