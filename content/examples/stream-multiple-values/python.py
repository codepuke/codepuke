PointSchema = Schema("Point", X=GOB_INT, Y=GOB_INT)

buf = io.BytesIO()
enc = Encoder(buf)
enc.encode({"X": 3, "Y": 4}, schema=PointSchema)
enc.encode({"X": 5, "Y": 6}, schema=PointSchema)

buf.seek(0)
dec = Decoder(buf)
first = dec.decode()
second = dec.decode()
