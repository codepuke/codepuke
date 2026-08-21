var buf bytes.Buffer
enc := gob.NewEncoder(&buf)
for _, p := range []Point{{1, 2}, {3, 4}, {5, 6}} {
	if err := enc.Encode(p); err != nil {
		log.Fatal(err)
	}
}

dec := gob.NewDecoder(&buf)
for {
	var p Point
	if err := dec.Decode(&p); err == io.EOF {
		break
	} else if err != nil {
		log.Fatal(err)
	}
	fmt.Println(p.X, p.Y)
}
