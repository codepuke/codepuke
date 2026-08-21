var p Point
dec := gob.NewDecoder(&buf)
if err := dec.Decode(&p); err != nil {
	log.Fatal(err)
}
fmt.Println("X =", p.X, "Y =", p.Y)
