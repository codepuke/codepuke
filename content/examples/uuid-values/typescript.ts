// uuid.UUID is a BinaryMarshaler, wire type name "UUID".
const enc = new GobEncoder({ codecs: DEFAULT_CODECS });
enc.encode('6ba7b810-9dad-11d1-80b4-00c04fd430c8', {
  marshalerType: 'UUID',
  marshalerKind: 'binary',
});

// Decodes to the canonical hyphenated lowercase form, matching the output of
// crypto.randomUUID().
const id = decode<string>(enc.bytes(), { codecs: DEFAULT_CODECS });
