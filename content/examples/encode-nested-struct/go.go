type Size struct{ W, H int }
type Photo struct {
	Title string
	Dim   Size
}

var buf bytes.Buffer
enc := gob.NewEncoder(&buf)
if err := enc.Encode(Photo{Title: "sunset", Dim: Size{W: 640, H: 480}}); err != nil {
	log.Fatal(err)
}
