primes := []int{2, 3, 5, 7, 11}

var buf bytes.Buffer
enc := gob.NewEncoder(&buf)
if err := enc.Encode(primes); err != nil {
	log.Fatal(err)
}
