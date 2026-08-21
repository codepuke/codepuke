// Register the concrete type so its name travels on the wire.
gob.Register(Dog{})

var buf bytes.Buffer
enc := gob.NewEncoder(&buf)
if err := enc.Encode(Pet{Companion: Dog{Name: "Rex"}}); err != nil {
	log.Fatal(err)
}
