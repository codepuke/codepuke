// Go's time.Time implements GobEncoder (not BinaryMarshaler) and arrives on
// the wire under the unqualified type name "Time".
const enc = new GobEncoder({ codecs: DEFAULT_CODECS });
enc.encode(new Date('2009-11-10T23:00:00.000Z'), {
  marshalerType: 'Time',
  marshalerKind: 'gob',
});

const when = decode<Date>(enc.bytes(), { codecs: DEFAULT_CODECS });
when.toISOString(); // "2009-11-10T23:00:00.000Z"

// Date holds milliseconds; Go holds nanoseconds. Sub-millisecond precision,
// the zone offset, and the zone name are all lost in the conversion.
