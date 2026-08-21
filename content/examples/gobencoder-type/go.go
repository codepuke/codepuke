type customToken struct{ payload []byte }

func (c customToken) GobEncode() ([]byte, error) { return c.payload, nil }
func (c *customToken) GobDecode(b []byte) error {
	c.payload = append([]byte(nil), b...)
	return nil
}
