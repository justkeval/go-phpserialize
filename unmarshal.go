package phpserialize

// Unmarshal parses the PHP serialized data and stores the result in the value
// pointed to by v.
//
// It is a convenience wrapper that first parses data into the intermediate
// [Value] representation with [Parse] and then decodes that Value into v with
// [Decode]. v must be a non-nil pointer. See [Decode] for the PHP-to-Go type
// mapping, including how PHP values are decoded into an empty interface.
func Unmarshal(data []byte, v any) error {
	val, err := Parse(data)
	if err != nil {
		return err
	}
	return Decode(val, v)
}
