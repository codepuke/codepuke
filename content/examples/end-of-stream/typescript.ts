const dec = new GobDecoder(stream);

// tryDecode() never throws at end of stream — it reports it.
const seen: unknown[] = [];
while (dec.hasMore()) {
  const result = dec.tryDecode();
  if (!result.ok) break;
  seen.push(result.value);
}
// seen holds Point{3, 4} and Point{5, 6}

// decode() throws instead. Feed more bytes and retry when the stream is live.
let exhausted = false;
try {
  dec.decode();
} catch (err) {
  if (err instanceof EndOfStreamError) {
    exhausted = true;
  }
}
