package phpserialize

import (
	"math"
	"reflect"
	"strings"
)

// Decode fills the destination pointed to by dst with the contents of the
// intermediate [Value] using reflection. It is the only decoding layer that
// inspects Go types; it never parses raw bytes.
//
// dst must be a non-nil pointer; otherwise an [InvalidUnmarshalError] is
// returned. Decoding into an interface value uses the default mapping described
// on [Unmarshal].
func Decode(v Value, dst any) error {
	rv := reflect.ValueOf(dst)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return &InvalidUnmarshalError{Type: reflect.TypeOf(dst)}
	}
	return decodeValue(v, rv.Elem(), "")
}

// decodeValue decodes v into the settable reflect.Value rv. field is a
// human-readable path used only for error messages.
func decodeValue(v Value, rv reflect.Value, field string) error {
	// Follow pointers, allocating as needed. A PHP null clears the pointer.
	if rv.Kind() == reflect.Pointer {
		if v.Kind == NullKind {
			rv.Set(reflect.Zero(rv.Type()))
			return nil
		}
		if rv.IsNil() {
			rv.Set(reflect.New(rv.Type().Elem()))
		}
		return decodeValue(v, rv.Elem(), field)
	}

	// An empty interface receives the default native mapping.
	if rv.Kind() == reflect.Interface {
		if rv.NumMethod() != 0 {
			return &UnmarshalTypeError{Value: v.Kind.String(), Type: rv.Type(), Field: field}
		}
		native, err := toNative(v)
		if err != nil {
			return err
		}
		if native == nil {
			rv.Set(reflect.Zero(rv.Type()))
		} else {
			rv.Set(reflect.ValueOf(native))
		}
		return nil
	}

	switch v.Kind {
	case NullKind:
		rv.Set(reflect.Zero(rv.Type()))
		return nil
	case BoolKind:
		return decodeBool(v, rv, field)
	case IntKind:
		return decodeInt(v, rv, field)
	case FloatKind:
		return decodeFloat(v, rv, field)
	case StringKind:
		return decodeString(v, rv, field)
	case ArrayKind:
		return decodeArray(v, rv, field)
	case ObjectKind:
		return decodeObject(v, rv, field)
	default:
		return &UnmarshalTypeError{Value: v.Kind.String(), Type: rv.Type(), Field: field}
	}
}

func decodeBool(v Value, rv reflect.Value, field string) error {
	if rv.Kind() == reflect.Bool {
		rv.SetBool(v.Bool)
		return nil
	}
	return &UnmarshalTypeError{Value: "bool", Type: rv.Type(), Field: field}
}

func decodeInt(v Value, rv reflect.Value, field string) error {
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if rv.OverflowInt(v.Int) {
			return &UnmarshalTypeError{Value: "int", Type: rv.Type(), Field: field}
		}
		rv.SetInt(v.Int)
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		if v.Int < 0 || rv.OverflowUint(uint64(v.Int)) {
			return &UnmarshalTypeError{Value: "int", Type: rv.Type(), Field: field}
		}
		rv.SetUint(uint64(v.Int))
		return nil
	case reflect.Float32, reflect.Float64:
		rv.SetFloat(float64(v.Int))
		return nil
	default:
		return &UnmarshalTypeError{Value: "int", Type: rv.Type(), Field: field}
	}
}

func decodeFloat(v Value, rv reflect.Value, field string) error {
	switch rv.Kind() {
	case reflect.Float32, reflect.Float64:
		if rv.OverflowFloat(v.Float) {
			return &UnmarshalTypeError{Value: "float", Type: rv.Type(), Field: field}
		}
		rv.SetFloat(v.Float)
		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// Allow a float that is exactly integral to fill an integer.
		if math.IsInf(v.Float, 0) || math.IsNaN(v.Float) || v.Float != math.Trunc(v.Float) {
			return &UnmarshalTypeError{Value: "float", Type: rv.Type(), Field: field}
		}
		n := int64(v.Float)
		if float64(n) != v.Float || rv.OverflowInt(n) {
			return &UnmarshalTypeError{Value: "float", Type: rv.Type(), Field: field}
		}
		rv.SetInt(n)
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		if math.IsInf(v.Float, 0) || math.IsNaN(v.Float) || v.Float != math.Trunc(v.Float) || v.Float < 0 {
			return &UnmarshalTypeError{Value: "float", Type: rv.Type(), Field: field}
		}
		u := uint64(v.Float)
		if float64(u) != v.Float || rv.OverflowUint(u) {
			return &UnmarshalTypeError{Value: "float", Type: rv.Type(), Field: field}
		}
		rv.SetUint(u)
		return nil
	default:
		return &UnmarshalTypeError{Value: "float", Type: rv.Type(), Field: field}
	}
}

func decodeString(v Value, rv reflect.Value, field string) error {
	switch rv.Kind() {
	case reflect.String:
		rv.SetString(v.String)
		return nil
	case reflect.Slice:
		if rv.Type().Elem().Kind() == reflect.Uint8 {
			rv.SetBytes([]byte(v.String))
			return nil
		}
	}
	return &UnmarshalTypeError{Value: "string", Type: rv.Type(), Field: field}
}

// decodeArray decodes a PHP array into a slice, array, map or struct.
func decodeArray(v Value, rv reflect.Value, field string) error {
	switch rv.Kind() {
	case reflect.Slice:
		return decodeSlice(v.Array, rv, field)
	case reflect.Array:
		return decodeToArray(v.Array, rv, field)
	case reflect.Map:
		return decodeMap(v.Array, rv, field)
	case reflect.Struct:
		return decodeStructEntries(v.Array, rv, field)
	default:
		return &UnmarshalTypeError{Value: "array", Type: rv.Type(), Field: field}
	}
}

func decodeSlice(entries []Entry, rv reflect.Value, field string) error {
	slice := reflect.MakeSlice(rv.Type(), len(entries), len(entries))
	for i, e := range entries {
		if err := decodeValue(e.Value, slice.Index(i), field); err != nil {
			return err
		}
	}
	rv.Set(slice)
	return nil
}

func decodeToArray(entries []Entry, rv reflect.Value, field string) error {
	n := rv.Len()
	for i := 0; i < n; i++ {
		if i < len(entries) {
			if err := decodeValue(entries[i].Value, rv.Index(i), field); err != nil {
				return err
			}
		} else {
			rv.Index(i).Set(reflect.Zero(rv.Type().Elem()))
		}
	}
	return nil
}

func decodeMap(entries []Entry, rv reflect.Value, field string) error {
	mt := rv.Type()
	if rv.IsNil() {
		rv.Set(reflect.MakeMapWithSize(mt, len(entries)))
	}
	keyType := mt.Key()
	elemType := mt.Elem()
	for _, e := range entries {
		kv := reflect.New(keyType).Elem()
		if err := decodeMapKey(e.Key, kv, field); err != nil {
			return err
		}
		ev := reflect.New(elemType).Elem()
		if err := decodeValue(e.Value, ev, field); err != nil {
			return err
		}
		rv.SetMapIndex(kv, ev)
	}
	return nil
}

// decodeMapKey decodes a PHP array key into a map key value. Integer keys can
// fill integer or string map keys; string keys fill string (or interface) map
// keys.
func decodeMapKey(key Value, kv reflect.Value, field string) error {
	if kv.Kind() == reflect.Interface && kv.NumMethod() == 0 {
		native, err := toNative(key)
		if err != nil {
			return err
		}
		kv.Set(reflect.ValueOf(native))
		return nil
	}
	switch key.Kind {
	case StringKind:
		if kv.Kind() == reflect.String {
			kv.SetString(key.String)
			return nil
		}
	case IntKind:
		return decodeInt(key, kv, field)
	}
	return &UnmarshalTypeError{Value: key.Kind.String(), Type: kv.Type(), Field: field}
}

// decodeObject decodes a PHP object into a struct, map or interface. The class
// name is discarded.
func decodeObject(v Value, rv reflect.Value, field string) error {
	obj := v.Object
	if obj == nil {
		obj = &Object{}
	}
	switch rv.Kind() {
	case reflect.Struct:
		return decodeStructFields(obj.Fields, rv, field)
	case reflect.Map:
		// Present object fields as string-keyed entries.
		entries := make([]Entry, len(obj.Fields))
		for i, f := range obj.Fields {
			entries[i] = Entry{Key: Value{Kind: StringKind, String: f.Name}, Value: f.Value}
		}
		return decodeMap(entries, rv, field)
	default:
		return &UnmarshalTypeError{Value: "object", Type: rv.Type(), Field: field}
	}
}

// decodeStructEntries decodes PHP array entries into a struct, matching only
// string keys against field names.
func decodeStructEntries(entries []Entry, rv reflect.Value, field string) error {
	fields := typeFields(rv.Type())
	byName, byLower := fieldIndex(fields)
	for _, e := range entries {
		if e.Key.Kind != StringKind {
			continue // integer-keyed entries cannot match named fields
		}
		if err := assignField(e.Key.String, e.Value, rv, fields, byName, byLower, field); err != nil {
			return err
		}
	}
	return nil
}

// decodeStructFields decodes PHP object fields into a struct.
func decodeStructFields(objFields []Field, rv reflect.Value, field string) error {
	fields := typeFields(rv.Type())
	byName, byLower := fieldIndex(fields)
	for _, f := range objFields {
		if err := assignField(f.Name, f.Value, rv, fields, byName, byLower, field); err != nil {
			return err
		}
	}
	return nil
}

// fieldIndex builds exact and case-insensitive lookup tables for a field list.
func fieldIndex(fields []field) (byName, byLower map[string]int) {
	byName = make(map[string]int, len(fields))
	byLower = make(map[string]int, len(fields))
	for i, f := range fields {
		byName[f.name] = i
		lower := strings.ToLower(f.name)
		if _, ok := byLower[lower]; !ok {
			byLower[lower] = i
		}
	}
	return byName, byLower
}

// assignField locates the struct field matching name and decodes val into it.
// Unknown fields are ignored. Exact name matches take precedence over
// case-insensitive ones.
func assignField(name string, val Value, rv reflect.Value, fields []field, byName, byLower map[string]int, path string) error {
	idx, ok := byName[name]
	if !ok {
		idx, ok = byLower[strings.ToLower(name)]
	}
	if !ok {
		return nil // unknown field: ignore
	}
	f := fields[idx]
	target := fieldByIndexAlloc(rv, f.index)
	childPath := f.name
	if path != "" {
		childPath = path + "." + f.name
	}
	return decodeValue(val, target, childPath)
}

// toNative converts a Value into its default Go representation for decoding
// into an empty interface, as documented on [Unmarshal].
func toNative(v Value) (any, error) {
	switch v.Kind {
	case NullKind:
		return nil, nil
	case BoolKind:
		return v.Bool, nil
	case IntKind:
		return v.Int, nil
	case FloatKind:
		return v.Float, nil
	case StringKind:
		return v.String, nil
	case ArrayKind:
		return arrayToNative(v.Array)
	case ObjectKind:
		obj := v.Object
		if obj == nil {
			obj = &Object{}
		}
		m := make(map[string]any, len(obj.Fields))
		for _, f := range obj.Fields {
			nv, err := toNative(f.Value)
			if err != nil {
				return nil, err
			}
			m[f.Name] = nv
		}
		return m, nil
	default:
		return nil, &UnmarshalTypeError{Value: v.Kind.String(), Type: nil}
	}
}

// arrayToNative converts a PHP array into either []any (when it has exactly the
// sequential integer keys 0..n-1) or map[any]any (otherwise).
func arrayToNative(entries []Entry) (any, error) {
	if isSequential(entries) {
		out := make([]any, len(entries))
		for i, e := range entries {
			nv, err := toNative(e.Value)
			if err != nil {
				return nil, err
			}
			out[i] = nv
		}
		return out, nil
	}
	out := make(map[any]any, len(entries))
	for _, e := range entries {
		var key any
		switch e.Key.Kind {
		case IntKind:
			key = e.Key.Int
		case StringKind:
			key = e.Key.String
		default:
			return nil, &UnmarshalTypeError{Value: e.Key.Kind.String(), Type: nil}
		}
		nv, err := toNative(e.Value)
		if err != nil {
			return nil, err
		}
		out[key] = nv
	}
	return out, nil
}

// isSequential reports whether entries have exactly the integer keys 0..n-1 in
// order, i.e. whether the PHP array is a plain list.
func isSequential(entries []Entry) bool {
	for i, e := range entries {
		if e.Key.Kind != IntKind || e.Key.Int != int64(i) {
			return false
		}
	}
	return true
}
