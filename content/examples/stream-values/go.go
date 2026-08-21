// Encode two values the usual way.
type User struct {
	Name string
	Age  int
}
var buf bytes.Buffer
enc := gob.NewEncoder(&buf)
enc.Encode(User{Name: "alice", Age: 30})
enc.Encode(User{Name: "bob", Age: 25})

// Decode the stream without the original type.
ins := gobspect.New()
for v, err := range ins.Stream(&buf).Values() {
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(gobspect.Format(v))
}
