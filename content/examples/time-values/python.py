# Go's time.Time implements GobEncoder (not BinaryMarshaler) and arrives
# on the wire under the unqualified type name "Time".
buf = io.BytesIO()
enc = Encoder(buf, codecs=DEFAULT_CODECS)
enc.encode_gob_encoded(datetime(2009, 11, 10, 23, 0, tzinfo=timezone.utc), "Time")

when = pygob.decode(buf.getvalue(), codecs=DEFAULT_CODECS)
print(when.isoformat())   # 2009-11-10T23:00:00+00:00

# Python's datetime holds microseconds; Go holds nanoseconds. Go values
# with sub-microsecond precision lose it in the conversion.
