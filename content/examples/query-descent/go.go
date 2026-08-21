type Server struct {
	Host   string
	Status string
	Tags   []string
}
type Cluster struct {
	Region  string
	Servers []Server
}
type Inventory struct {
	Clusters []Cluster
}

var buf bytes.Buffer
enc := gob.NewEncoder(&buf)
if err := enc.Encode(Inventory{Clusters: []Cluster{
	{Region: "eu-west", Servers: []Server{
		{Host: "web-1", Status: "active", Tags: []string{"web", "prod"}},
		{Host: "db-1", Status: "draining", Tags: []string{"db", "prod"}},
	}},
	{Region: "us-east", Servers: []Server{
		{Host: "web-2", Status: "active", Tags: []string{"web", "staging"}},
		{Host: "cache-1", Status: "active", Tags: []string{"cache", "prod"}},
	}},
}}); err != nil {
	log.Fatal(err)
}

values, err := gobspect.New().Stream(&buf).Collect()
if err != nil {
	log.Fatal(err)
}
root := values[0]

// "..Host" collects every Host field at any depth.
for _, v := range query.All(root, "..Host") {
	fmt.Println(gobspect.Format(v))
}

// Wildcard descent with filters: keep nodes tagged "prod" whose
// Status is "active", then project just the fields we care about.
for _, v := range query.All(root, "..[Tags~prod][Status=active].Host,Status") {
	fmt.Println(gobspect.Format(v))
}
