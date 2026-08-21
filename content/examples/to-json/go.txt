type Point struct {
	X, Y int
}
var buf bytes.Buffer
gob.NewEncoder(&buf).Encode(Point{X: 3, Y: 7})

ins := gobspect.New()
values, err := ins.Stream(&buf).Collect()
if err != nil {
	log.Fatal(err)
}

// Type IDs are scoped to the stream that produced the value; zero the
// ID here so the JSON is reproducible.
sv := values[0].(gobspect.StructValue)
sv.GobTypeID = 0

b, err := gobspect.ToJSONIndent(sv, "", "  ")
if err != nil {
	log.Fatal(err)
}
fmt.Println(string(b))
