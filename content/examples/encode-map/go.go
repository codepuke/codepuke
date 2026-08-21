counts := map[string]int{"apples": 3, "pears": 7}

var buf bytes.Buffer
enc := gob.NewEncoder(&buf)
if err := enc.Encode(counts); err != nil {
	log.Fatal(err)
}
