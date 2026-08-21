pygob.encode(42)             # int      → signed int
pygob.encode(UInt(42))       # UInt     → unsigned int
pygob.encode(3.14159)        # float    → float64
pygob.encode(True)           # bool     → bool
pygob.encode("hello, 世界")   # str      → string
pygob.encode(b"raw")         # bytes    → []byte
pygob.encode(1 + 2j)         # complex  → complex128
