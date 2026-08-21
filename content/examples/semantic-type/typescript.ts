// Go: type Status string
type Status = 'active' | 'inactive';

const GOB_STATUS = SemanticType<Status>({
  wire: GOB_STRING,
  encode: (value) => value,
  decode: (wire) => wire as Status,
  zero: 'inactive',
});

const UserSchema = new Schema('User', {
  Name: GOB_STRING,
  Status: GOB_STATUS,
});

const bytes = encode({ Name: 'Ada', Status: 'active' }, { schema: UserSchema });
const user = decode<GobObject>(bytes);
const status = user.get('Status') as Status; // "active"
