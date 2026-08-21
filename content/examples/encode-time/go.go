launch := time.Date(2024, time.March, 14, 15, 9, 26, 0, time.UTC)

var buf bytes.Buffer
if err := gob.NewEncoder(&buf).Encode(launch); err != nil {
	log.Fatal(err)
}

var decoded time.Time
if err := gob.NewDecoder(&buf).Decode(&decoded); err != nil {
	log.Fatal(err)
}
fmt.Println(decoded.Format(time.RFC3339))
