PointSchema = Schema("Point", X=GOB_INT, Y=GOB_INT)

full = pygob.encode({"X": 3, "Y": 4}, schema=PointSchema)
partial = pygob.encode({"X": 3, "Y": 0}, schema=PointSchema)

# Y == 0 is simply absent from the encoded bytes...
assert len(partial) < len(full)

# ...but the decoder still fills it in.
assert pygob.decode(partial).Y == 0
