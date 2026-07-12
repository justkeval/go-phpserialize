package phpserialize

import (
	"math"
	"reflect"
	"sort"
)

// PHPClass may be implemented by a Go type to control how it is encoded. When a
// value's type implements PHPClass, [Encode] serializes it as a PHP object
// ("O") using the returned class name, rather than as an associative array.
type PHPClass interface {
	// PHPClassName returns the PHP class name to use when serializing the
	// implementing value as a PHP object.
	PHPClassName() string
}

// Encode converts a Go value into the intermediate [Value] representation using
// reflection. It is the only encoding layer that inspects Go types; the writer
// operates purely on [Value].
//
// The mapping is:
//
//   - nil, nil pointers and nil interfaces -> PHP null.
//   - bool -> PHP bool.
//   - all integer and unsigned integer types -> PHP int.
//   - float32/float64 -> PHP double.
//   - string -> PHP string.
//   - []byte -> PHP string (raw bytes).
//   - slices and arrays -> sequential PHP array (keys 0..n-1).
//   - maps -> PHP array keyed by the map keys (keys sorted for determinism).
//   - structs -> associative PHP array, or a PHP object if the type
//     implements [PHPClass].
//   - pointers and interfaces -> the encoding of the value they refer to.
func Encode(v any) (Value, error) {
	if v == nil {
		return Value{Kind: NullKind}, nil
	}
	return encodeValue(reflect.ValueOf(v))
}

// encodeValue encodes a single reflect.Value.
func encodeValue(v reflect.Value) (Value, error) {
	// Detect a PHPClass implementation before unwrapping so pointer receivers
	// are honored.
	if name, ok := phpClassName(v); ok {
		return encodeAsObject(v, name)
	}

	switch v.Kind() {
	case reflect.Invalid:
		return Value{Kind: NullKind}, nil

	case reflect.Bool:
		return Value{Kind: BoolKind, Bool: v.Bool()}, nil

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return Value{Kind: IntKind, Int: v.Int()}, nil

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		u := v.Uint()
		if u > math.MaxInt64 {
			return Value{}, &UnsupportedValueError{Value: v, Str: "unsigned integer overflows int64"}
		}
		return Value{Kind: IntKind, Int: int64(u)}, nil

	case reflect.Float32, reflect.Float64:
		return Value{Kind: FloatKind, Float: v.Float()}, nil

	case reflect.String:
		return Value{Kind: StringKind, String: v.String()}, nil

	case reflect.Pointer, reflect.Interface:
		if v.IsNil() {
			return Value{Kind: NullKind}, nil
		}
		return encodeValue(v.Elem())

	case reflect.Slice:
		if v.IsNil() {
			return Value{Kind: NullKind}, nil
		}
		if v.Type().Elem().Kind() == reflect.Uint8 {
			return Value{Kind: StringKind, String: string(v.Bytes())}, nil
		}
		return encodeSequence(v)

	case reflect.Array:
		return encodeSequence(v)

	case reflect.Map:
		if v.IsNil() {
			return Value{Kind: NullKind}, nil
		}
		return encodeMap(v)

	case reflect.Struct:
		return encodeStruct(v)

	default:
		return Value{}, &UnsupportedTypeError{Type: v.Type()}
	}
}

// encodeSequence encodes a slice or array as a sequential PHP array with
// integer keys 0..n-1.
func encodeSequence(v reflect.Value) (Value, error) {
	n := v.Len()
	entries := make([]Entry, 0, n)
	for i := 0; i < n; i++ {
		elem, err := encodeValue(v.Index(i))
		if err != nil {
			return Value{}, err
		}
		entries = append(entries, Entry{
			Key:   Value{Kind: IntKind, Int: int64(i)},
			Value: elem,
		})
	}
	return Value{Kind: ArrayKind, Array: entries}, nil
}

// encodeMap encodes a Go map as a PHP array. Keys are emitted in a
// deterministic order: numerically for integer keys, lexically for string
// keys.
func encodeMap(v reflect.Value) (Value, error) {
	keys := v.MapKeys()
	encKeys := make([]Value, len(keys))
	for i, k := range keys {
		ek, err := encodeMapKey(k)
		if err != nil {
			return Value{}, err
		}
		encKeys[i] = ek
	}

	order := make([]int, len(keys))
	for i := range order {
		order[i] = i
	}
	sort.Slice(order, func(a, b int) bool {
		ka, kb := encKeys[order[a]], encKeys[order[b]]
		if ka.Kind != kb.Kind {
			return ka.Kind < kb.Kind
		}
		if ka.Kind == IntKind {
			return ka.Int < kb.Int
		}
		return ka.String < kb.String
	})

	entries := make([]Entry, 0, len(keys))
	for _, idx := range order {
		val, err := encodeValue(v.MapIndex(keys[idx]))
		if err != nil {
			return Value{}, err
		}
		entries = append(entries, Entry{Key: encKeys[idx], Value: val})
	}
	return Value{Kind: ArrayKind, Array: entries}, nil
}

// encodeMapKey converts a map key into a PHP array key (int or string).
// Interface and pointer keys are unwrapped so that maps such as map[any]any
// round-trip correctly.
func encodeMapKey(k reflect.Value) (Value, error) {
	for k.Kind() == reflect.Interface || k.Kind() == reflect.Pointer {
		if k.IsNil() {
			return Value{}, &UnsupportedValueError{Value: k, Str: "nil map key"}
		}
		k = k.Elem()
	}
	switch k.Kind() {
	case reflect.String:
		return Value{Kind: StringKind, String: k.String()}, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return Value{Kind: IntKind, Int: k.Int()}, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		u := k.Uint()
		if u > math.MaxInt64 {
			return Value{}, &UnsupportedValueError{Value: k, Str: "map key overflows int64"}
		}
		return Value{Kind: IntKind, Int: int64(u)}, nil
	default:
		return Value{}, &UnsupportedTypeError{Type: k.Type()}
	}
}

// encodeStruct encodes a struct as an associative PHP array.
func encodeStruct(v reflect.Value) (Value, error) {
	fields := typeFields(v.Type())
	entries := make([]Entry, 0, len(fields))
	for i := range fields {
		f := &fields[i]
		fv := fieldByIndex(v, f.index)
		if !fv.IsValid() {
			continue
		}
		if f.omitEmpty && isEmptyValue(fv) {
			continue
		}
		ev, err := encodeValue(fv)
		if err != nil {
			return Value{}, err
		}
		entries = append(entries, Entry{
			Key:   Value{Kind: StringKind, String: f.name},
			Value: ev,
		})
	}
	return Value{Kind: ArrayKind, Array: entries}, nil
}

// encodeAsObject encodes a value whose type implements [PHPClass] as a PHP
// object. Only structs carry named fields; other kinds produce an object with
// no fields but the requested class name.
func encodeAsObject(v reflect.Value, className string) (Value, error) {
	// Unwrap pointers/interfaces to reach the underlying struct.
	uv := v
	for uv.Kind() == reflect.Pointer || uv.Kind() == reflect.Interface {
		if uv.IsNil() {
			return Value{Kind: NullKind}, nil
		}
		uv = uv.Elem()
	}
	obj := &Object{ClassName: className}
	if uv.Kind() == reflect.Struct {
		fields := typeFields(uv.Type())
		for i := range fields {
			f := &fields[i]
			fv := fieldByIndex(uv, f.index)
			if !fv.IsValid() {
				continue
			}
			if f.omitEmpty && isEmptyValue(fv) {
				continue
			}
			ev, err := encodeValue(fv)
			if err != nil {
				return Value{}, err
			}
			obj.Fields = append(obj.Fields, Field{Name: f.name, Value: ev})
		}
	}
	return Value{Kind: ObjectKind, Object: obj}, nil
}

// phpClassName reports the PHP class name for v if its type (or its pointer
// type, when addressable) implements [PHPClass].
func phpClassName(v reflect.Value) (string, bool) {
	if !v.IsValid() || !v.CanInterface() {
		return "", false
	}
	if v.Kind() == reflect.Pointer && v.IsNil() {
		return "", false
	}
	if pc, ok := v.Interface().(PHPClass); ok {
		return pc.PHPClassName(), true
	}
	if v.Kind() != reflect.Pointer && v.CanAddr() {
		if pc, ok := v.Addr().Interface().(PHPClass); ok {
			return pc.PHPClassName(), true
		}
	}
	return "", false
}

// isEmptyValue reports whether v is the zero value for its type, using the same
// definition as encoding/json's omitempty.
func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Pointer:
		return v.IsNil()
	}
	return false
}
