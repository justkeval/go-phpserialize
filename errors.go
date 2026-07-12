package phpserialize

import (
	"errors"
	"reflect"
	"strconv"
)

// ErrUnsupportedType is returned by the parser when it encounters a PHP
// serialized type that this package deliberately does not support: custom
// serialized objects ("C") and references ("R"/"r").
var ErrUnsupportedType = errors.New("phpserialize: unsupported type")

// SyntaxError describes malformed PHP serialized input. Offset records the
// byte position in the input where the error was detected.
type SyntaxError struct {
	msg    string
	Offset int64
}

func (e *SyntaxError) Error() string {
	return "phpserialize: " + e.msg + " at offset " + strconv.FormatInt(e.Offset, 10)
}

// UnmarshalTypeError describes a PHP value that was not appropriate for a value
// of a specific Go type during decoding.
type UnmarshalTypeError struct {
	// Value is a description of the PHP value ("int", "string", ...).
	Value string
	// Type is the Go type that could not hold the PHP value.
	Type reflect.Type
	// Field is the struct field path being decoded, if any.
	Field string
}

func (e *UnmarshalTypeError) Error() string {
	if e.Field != "" {
		return "phpserialize: cannot unmarshal PHP " + e.Value +
			" into Go struct field " + e.Field +
			" of type " + e.Type.String()
	}
	return "phpserialize: cannot unmarshal PHP " + e.Value +
		" into Go value of type " + e.Type.String()
}

// InvalidUnmarshalError describes an invalid argument passed to [Unmarshal] or
// [Decode]. The argument must be a non-nil pointer.
type InvalidUnmarshalError struct {
	Type reflect.Type
}

func (e *InvalidUnmarshalError) Error() string {
	if e.Type == nil {
		return "phpserialize: Unmarshal(nil)"
	}
	if e.Type.Kind() != reflect.Pointer {
		return "phpserialize: Unmarshal(non-pointer " + e.Type.String() + ")"
	}
	return "phpserialize: Unmarshal(nil " + e.Type.String() + ")"
}

// UnsupportedTypeError is returned by [Encode] and [Marshal] when a Go value of
// a type that cannot be represented in the PHP serialized format is
// encountered (for example a channel or function).
type UnsupportedTypeError struct {
	Type reflect.Type
}

func (e *UnsupportedTypeError) Error() string {
	return "phpserialize: unsupported type: " + e.Type.String()
}

// UnsupportedValueError is returned by [Encode] and [Marshal] when a Go value
// cannot be represented in the PHP serialized format, such as a map with an
// unusable key type.
type UnsupportedValueError struct {
	Value reflect.Value
	Str   string
}

func (e *UnsupportedValueError) Error() string {
	return "phpserialize: unsupported value: " + e.Str
}
