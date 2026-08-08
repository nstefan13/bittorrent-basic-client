package bencoding

// Intermediate bencoding types representing decoded values before they are mapped to concrete domain structs.
type (
	// BInt represents a bencoded integer (e.g. "i42e").
	BInt int

	// BString represents a bencoded byte string (e.g. "4:spam").
	BString string

	// BList represents a bencoded list of heterogeneous elements (e.g. "l4:spami42ee").
	BList []any

	// BDict represents a bencoded dictionary mapping string keys to arbitrary bencoded values (e.g. "d3:cow3:mooe").
	BDict map[string]any
)

