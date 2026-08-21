// sessionToken implements GobEncoder; wrap it in an interface field so
// its type name travels on the wire.
type Envelope struct{ Token any }
gob.RegisterName("sessionToken", sessionToken{})

var buf bytes.Buffer
gob.NewEncoder(&buf).Encode(Envelope{Token: sessionToken{payload: []byte("tok-12345")}})

ins := gobspect.New()
// The key is the type's name from the gob wire format.
ins.RegisterDecoder("sessionToken", func(data []byte) (any, error) {
	return "session:" + string(data), nil
})

values, err := ins.Stream(&buf).Collect()
if err != nil {
	log.Fatal(err)
}
fmt.Println(gobspect.Format(values[0]))
