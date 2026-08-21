dec = Decoder(buf)
records = []
try:
    while True:
        records.append(dec.decode())
except GobDecodeError:
    pass    # stream exhausted
