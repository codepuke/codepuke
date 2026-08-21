PointSchema = Schema("Point", X=GOB_INT, Y=GOB_INT)
data = pygob.encode({"X": 3, "Y": 4}, schema=PointSchema)
