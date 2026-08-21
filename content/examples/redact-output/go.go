type Login struct {
	Username string
	Password string
}
var buf bytes.Buffer
gob.NewEncoder(&buf).Encode(Login{Username: "alice", Password: "hunter2"})

ins := gobspect.New()
values, err := ins.Stream(&buf).Collect()
if err != nil {
	log.Fatal(err)
}

// Redact by field or map-key name.
fmt.Println(gobspect.Format(values[0],
	gobspect.WithRedactKeys(gobspect.RedactConfig{Keys: []string{"Password"}}),
))

// Redact every value of a named type.
fmt.Println(gobspect.Format(values[0],
	gobspect.WithRedactTypes(gobspect.RedactTypesConfig{Types: []string{"Login"}, TextLength: 3}),
))
