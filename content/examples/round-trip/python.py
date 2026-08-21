PointSchema = Schema("Point", X=GOB_INT, Y=GOB_INT)

data = pygob.encode({"X": 3, "Y": 4}, schema=PointSchema)
point = pygob.decode(data)

assert point.X == 3
assert point.Y == 4
