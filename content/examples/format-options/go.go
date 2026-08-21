// Default rendering.
fmt.Println(gobspect.Format(v))

// Wider indentation and base64-encoded byte slices.
fmt.Println(gobspect.Format(v,
	gobspect.WithIndent("    "),
	gobspect.WithBytesFormat(gobspect.BytesBase64),
))

// Go-literal byte rendering.
fmt.Println(gobspect.Format(v, gobspect.WithBytesFormat(gobspect.BytesLiteral)))
