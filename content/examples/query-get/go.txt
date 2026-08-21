type Customer struct {
	Name string
}
type Order struct {
	ID       int
	Total    float64
	Customer Customer
}
type Report struct {
	Orders []Order
}

// Encode a gob stream the usual way.
var buf bytes.Buffer
enc := gob.NewEncoder(&buf)
if err := enc.Encode(Report{Orders: []Order{
	{ID: 1, Total: 9.5, Customer: Customer{Name: "Ada"}},
	{ID: 2, Total: 24, Customer: Customer{Name: "Grace"}},
	{ID: 3, Total: 12.25, Customer: Customer{Name: "Linus"}},
}}); err != nil {
	log.Fatal(err)
}

// Decode it with gobspect — no original types needed.
values, err := gobspect.New().Stream(&buf).Collect()
if err != nil {
	log.Fatal(err)
}
root := values[0]

// Dot paths navigate struct fields and slice indexes.
name, _ := query.Get(root, "Orders.0.Customer.Name")
fmt.Println(gobspect.Format(name))

// Negative indexes count from the end: -1 is the last element.
last, _ := query.Get(root, "Orders.-1.Customer.Name")
fmt.Println(gobspect.Format(last))

// A wildcard fans out across every element.
for _, v := range query.All(root, "Orders.*.Total") {
	fmt.Println(gobspect.Format(v))
}
