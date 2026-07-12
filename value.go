// Package phpserialize implements serialization and deserialization of the
// PHP serialize() data format.
//
// The package is organized into four independent layers:
//
//   - Parser (parser.go): PHP serialized bytes -> Value
//   - Writer (writer.go): Value -> PHP serialized bytes
//   - Encode (encode.go): Go value -> Value (via reflection)
//   - Decode (decode.go): Value -> Go value (via reflection)
//
// The [Value] type is the intermediate representation shared by all four
// layers. The parser and writer never inspect Go destination types, and the
// encoder and decoder never touch raw bytes; reflection lives only in the
// encode and decode layers.
//
// The high-level entry points [Marshal] and [Unmarshal] mirror the API of the
// standard library's encoding/json package.
package phpserialize

// Kind identifies the PHP type carried by a [Value].
type Kind uint8

// The set of PHP types representable by a [Value].
const (
	// NullKind represents the PHP null value ("N;").
	NullKind Kind = iota
	// BoolKind represents a PHP boolean ("b:0;" or "b:1;").
	BoolKind
	// IntKind represents a PHP integer ("i:<n>;").
	IntKind
	// FloatKind represents a PHP double ("d:<n>;").
	FloatKind
	// StringKind represents a PHP string ("s:<len>:\"...\";").
	StringKind
	// ArrayKind represents a PHP array ("a:<n>:{...}").
	ArrayKind
	// ObjectKind represents a PHP object ("O:<len>:\"class\":<n>:{...}").
	ObjectKind
)

// String returns the name of the kind, useful in error messages.
func (k Kind) String() string {
	switch k {
	case NullKind:
		return "null"
	case BoolKind:
		return "bool"
	case IntKind:
		return "int"
	case FloatKind:
		return "float"
	case StringKind:
		return "string"
	case ArrayKind:
		return "array"
	case ObjectKind:
		return "object"
	default:
		return "unknown"
	}
}

// Value is the intermediate representation of a single PHP value. It is a
// tagged union: the Kind field selects which of the remaining fields carries
// meaningful data.
//
//   - NullKind:   no payload.
//   - BoolKind:   Bool.
//   - IntKind:    Int.
//   - FloatKind:  Float.
//   - StringKind: String.
//   - ArrayKind:  Array (ordered key/value entries).
//   - ObjectKind: Object (class name plus ordered fields).
type Value struct {
	Kind Kind

	Bool   bool
	Int    int64
	Float  float64
	String string

	Array  []Entry
	Object *Object
}

// Entry is a single ordered key/value pair within a PHP array.
type Entry struct {
	Key   Value
	Value Value
}

// Object is the payload of an [ObjectKind] value. It records the PHP class
// name and the object's fields in serialization order.
type Object struct {
	ClassName string
	Fields    []Field
}

// Field is a single named property of a PHP object, in serialization order.
type Field struct {
	Name  string
	Value Value
}

// Null returns a Value representing PHP null.
func Null() Value { return Value{Kind: NullKind} }

// Bool returns a Value representing a PHP boolean.
func Bool(b bool) Value { return Value{Kind: BoolKind, Bool: b} }

// Int returns a Value representing a PHP integer.
func Int(i int64) Value { return Value{Kind: IntKind, Int: i} }

// Float returns a Value representing a PHP double.
func Float(f float64) Value { return Value{Kind: FloatKind, Float: f} }

// String returns a Value representing a PHP string.
func String(s string) Value { return Value{Kind: StringKind, String: s} }

// Equal reports whether v and other represent the same PHP value. Two arrays
// are equal only when their entries match pairwise in order, and two objects
// only when their class names and ordered fields match.
func (v Value) Equal(other Value) bool {
	if v.Kind != other.Kind {
		return false
	}
	switch v.Kind {
	case NullKind:
		return true
	case BoolKind:
		return v.Bool == other.Bool
	case IntKind:
		return v.Int == other.Int
	case FloatKind:
		// Treat NaN as equal to NaN so round-trip tests are stable.
		if v.Float != v.Float && other.Float != other.Float {
			return true
		}
		return v.Float == other.Float
	case StringKind:
		return v.String == other.String
	case ArrayKind:
		if len(v.Array) != len(other.Array) {
			return false
		}
		for i := range v.Array {
			if !v.Array[i].Key.Equal(other.Array[i].Key) {
				return false
			}
			if !v.Array[i].Value.Equal(other.Array[i].Value) {
				return false
			}
		}
		return true
	case ObjectKind:
		if v.Object == nil || other.Object == nil {
			return v.Object == other.Object
		}
		if v.Object.ClassName != other.Object.ClassName {
			return false
		}
		if len(v.Object.Fields) != len(other.Object.Fields) {
			return false
		}
		for i := range v.Object.Fields {
			if v.Object.Fields[i].Name != other.Object.Fields[i].Name {
				return false
			}
			if !v.Object.Fields[i].Value.Equal(other.Object.Fields[i].Value) {
				return false
			}
		}
		return true
	default:
		return false
	}
}
