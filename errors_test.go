package phpserialize

import (
	"reflect"
	"strings"
	"testing"
)

func TestSyntaxErrorMessage(t *testing.T) {
	e := &SyntaxError{msg: "boom", Offset: 5}
	got := e.Error()
	if !strings.Contains(got, "boom") || !strings.Contains(got, "5") {
		t.Errorf("Error() = %q", got)
	}
}

func TestUnmarshalTypeErrorMessage(t *testing.T) {
	e := &UnmarshalTypeError{Value: "int", Type: reflect.TypeOf("")}
	if !strings.Contains(e.Error(), "int") || !strings.Contains(e.Error(), "string") {
		t.Errorf("Error() = %q", e.Error())
	}
	ef := &UnmarshalTypeError{Value: "int", Type: reflect.TypeOf(""), Field: "Foo.Bar"}
	if !strings.Contains(ef.Error(), "Foo.Bar") {
		t.Errorf("Error() = %q", ef.Error())
	}
}

func TestInvalidUnmarshalErrorMessage(t *testing.T) {
	if got := (&InvalidUnmarshalError{}).Error(); !strings.Contains(got, "nil") {
		t.Errorf("nil type: %q", got)
	}
	if got := (&InvalidUnmarshalError{Type: reflect.TypeOf(0)}).Error(); !strings.Contains(got, "non-pointer") {
		t.Errorf("non-pointer: %q", got)
	}
	if got := (&InvalidUnmarshalError{Type: reflect.TypeOf((*int)(nil))}).Error(); !strings.Contains(got, "nil") {
		t.Errorf("nil pointer: %q", got)
	}
}

func TestUnsupportedTypeErrorMessage(t *testing.T) {
	e := &UnsupportedTypeError{Type: reflect.TypeOf(make(chan int))}
	if !strings.Contains(e.Error(), "chan") {
		t.Errorf("Error() = %q", e.Error())
	}
}

func TestUnsupportedValueErrorMessage(t *testing.T) {
	e := &UnsupportedValueError{Str: "boom"}
	if !strings.Contains(e.Error(), "boom") {
		t.Errorf("Error() = %q", e.Error())
	}
}
