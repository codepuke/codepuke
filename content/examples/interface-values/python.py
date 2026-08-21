# Go: type Box struct { Value any }
PointSchema = Schema("Point", X=GOB_INT, Y=GOB_INT)
BoxSchema = Schema("Box", Value=GOB_INTERFACE)

# An interface value travels with its concrete type name, which Go
# qualifies by package — "main.Point", not "Point". Register that name;
# it must match whatever the Go side passed to gob.Register.
buf = io.BytesIO()
enc = Encoder(buf)
enc.register("main.Point", PointSchema)

point = GobStruct("Point", PointSchema, X=3, Y=4)
enc.encode({"Value": point}, schema=BoxSchema)

buf.seek(0)
box = Decoder(buf).decode()
print(box.Value.X)   # 3
