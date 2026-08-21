type Point struct {
	X, Y int
}
var buf bytes.Buffer
gob.NewEncoder(&buf).Encode(Point{X: 1, Y: 2})

ins := gobspect.New()
s := ins.Stream(&buf)
if _, err := s.Collect(); err != nil {
	log.Fatal(err)
}

// s.Types() now holds every type definition the stream declared.
for _, ti := range s.Types() {
	fmt.Printf("%s (%s)\n", ti.Name, ti.Kind)
	for _, f := range ti.Fields {
		fmt.Printf("  field %s\n", f.Name)
	}
}
