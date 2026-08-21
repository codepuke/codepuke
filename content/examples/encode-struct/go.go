var buf bytes.Buffer
enc := gob.NewEncoder(&buf)
if err := enc.Encode(Point{X: 3, Y: 4}); err != nil {
	log.Fatal(err)
}
