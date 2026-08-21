point = pygob.decode(data)

print(point.X)           # 3    — attribute access
print(point["Y"])        # 4    — item access
print(point.gob_type)    # "Point" — the Go type name
print(dict(point))       # {"X": 3, "Y": 4}
