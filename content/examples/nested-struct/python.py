# Go: type Line struct { From, To Point }
PointSchema = Schema("Point", X=GOB_INT, Y=GOB_INT)

# A Schema is itself a field type, so it nests directly.
LineSchema = Schema("Line", From=PointSchema, To=PointSchema)

data = pygob.encode(
    {"From": {"X": 1, "Y": 2}, "To": {"X": 3, "Y": 4}},
    schema=LineSchema,
)

line = pygob.decode(data)
print(line.To.X)   # 3
