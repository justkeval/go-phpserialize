package phpserialize

// Marshal returns the PHP serialized encoding of v.
//
// It is a convenience wrapper that first encodes v into the intermediate
// [Value] representation with [Encode] and then serializes that Value to bytes
// with [Write]. See [Encode] for the Go-to-PHP type mapping.
func Marshal(v any) ([]byte, error) {
	val, err := Encode(v)
	if err != nil {
		return nil, err
	}
	return Write(val)
}
