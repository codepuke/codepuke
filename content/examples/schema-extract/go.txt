type Customer struct {
	Name string
}
type Order struct {
	ID       int
	Customer Customer
	Tags     []string
}
var buf bytes.Buffer
gob.NewEncoder(&buf).Encode(Order{
	ID:       7,
	Customer: Customer{Name: "alice"},
	Tags:     []string{"rush"},
})

ins := gobspect.New()
schema, err := ins.Stream(&buf).Schema()
if err != nil {
	log.Fatal(err)
}
fmt.Println(schema.String())
