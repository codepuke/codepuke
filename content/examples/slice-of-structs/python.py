PointSchema = Schema("Point", X=GOB_INT, Y=GOB_INT)

data = pygob.encode(
    [{"X": 3, "Y": 4}, {"X": 5, "Y": 6}],
    elem_type=PointSchema,
)
