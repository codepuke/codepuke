const PersonSchema = new Schema('Person', {
  Name: GOB_STRING,
  Age: GOB_INT,
});

// InferSchema is type-level only — it emits no runtime code.
//   { Name: string; Age: bigint }
type Person = InferSchema<typeof PersonSchema>;

const ada: Person = { Name: 'Ada', Age: 36n };
const bytes = encode(ada, { schema: PersonSchema });
