package phpserialize

import (
	"errors"
	"math"
	"reflect"
	"testing"
)

func TestDecodePrimitivesTyped(t *testing.T) {
	var b bool
	if err := Unmarshal([]byte("b:1;"), &b); err != nil || b != true {
		t.Errorf("bool: %v %v", b, err)
	}
	var i int
	if err := Unmarshal([]byte("i:42;"), &i); err != nil || i != 42 {
		t.Errorf("int: %v %v", i, err)
	}
	var u uint8
	if err := Unmarshal([]byte("i:200;"), &u); err != nil || u != 200 {
		t.Errorf("uint8: %v %v", u, err)
	}
	var f float64
	if err := Unmarshal([]byte("d:2.5;"), &f); err != nil || f != 2.5 {
		t.Errorf("float: %v %v", f, err)
	}
	var s string
	if err := Unmarshal([]byte(`s:3:"abc";`), &s); err != nil || s != "abc" {
		t.Errorf("string: %v %v", s, err)
	}
	var by []byte
	if err := Unmarshal([]byte(`s:3:"abc";`), &by); err != nil || string(by) != "abc" {
		t.Errorf("bytes: %v %v", by, err)
	}
}

func TestDecodeNumericConversions(t *testing.T) {
	var i32 int32
	if err := Unmarshal([]byte("i:100;"), &i32); err != nil || i32 != 100 {
		t.Errorf("int32: %v %v", i32, err)
	}
	var f32 float32
	if err := Unmarshal([]byte("d:1.5;"), &f32); err != nil || f32 != 1.5 {
		t.Errorf("float32: %v %v", f32, err)
	}
	// int -> float
	var f float64
	if err := Unmarshal([]byte("i:7;"), &f); err != nil || f != 7 {
		t.Errorf("int->float: %v %v", f, err)
	}
	// integral float -> int
	var i int
	if err := Unmarshal([]byte("d:9;"), &i); err != nil || i != 9 {
		t.Errorf("float->int: %v %v", i, err)
	}
}

func TestDecodeNumericOverflow(t *testing.T) {
	var i8 int8
	err := Unmarshal([]byte("i:1000;"), &i8)
	var ute *UnmarshalTypeError
	if !errors.As(err, &ute) {
		t.Errorf("int8 overflow: err = %v, want UnmarshalTypeError", err)
	}
	var u uint8
	if err := Unmarshal([]byte("i:-1;"), &u); err == nil {
		t.Error("negative into uint should fail")
	}
	var f32 float32
	if err := Unmarshal([]byte("d:1.0E+40;"), &f32); err == nil {
		t.Error("float32 overflow should fail")
	}
	// non-integral float into int should fail
	var i int
	if err := Unmarshal([]byte("d:1.5;"), &i); err == nil {
		t.Error("non-integral float into int should fail")
	}
}

func TestDecodeTypeMismatch(t *testing.T) {
	var i int
	if err := Unmarshal([]byte(`s:1:"x";`), &i); err == nil {
		t.Error("string into int should fail")
	}
	var s string
	if err := Unmarshal([]byte("i:1;"), &s); err == nil {
		t.Error("int into string should fail")
	}
	var b bool
	if err := Unmarshal([]byte("i:1;"), &b); err == nil {
		t.Error("int into bool should fail")
	}
}

func TestDecodeNull(t *testing.T) {
	i := 5
	if err := Unmarshal([]byte("N;"), &i); err != nil || i != 0 {
		t.Errorf("null into int: %v %v", i, err)
	}
	p := new(int)
	if err := Unmarshal([]byte("N;"), &p); err != nil || p != nil {
		t.Errorf("null into *int: %v %v", p, err)
	}
	var iface any = "x"
	if err := Unmarshal([]byte("N;"), &iface); err != nil || iface != nil {
		t.Errorf("null into any: %v %v", iface, err)
	}
}

func TestDecodeSlice(t *testing.T) {
	var s []int
	if err := Unmarshal([]byte("a:3:{i:0;i:1;i:1;i:2;i:2;i:3;}"), &s); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(s, []int{1, 2, 3}) {
		t.Errorf("got %v", s)
	}
}

func TestDecodeArray(t *testing.T) {
	var a [2]int
	if err := Unmarshal([]byte("a:3:{i:0;i:1;i:1;i:2;i:2;i:3;}"), &a); err != nil {
		t.Fatal(err)
	}
	if a != [2]int{1, 2} {
		t.Errorf("got %v, want [1 2]", a)
	}
	// Fewer elements than array length: rest stays zero.
	var b [3]int
	if err := Unmarshal([]byte("a:1:{i:0;i:9;}"), &b); err != nil {
		t.Fatal(err)
	}
	if b != [3]int{9, 0, 0} {
		t.Errorf("got %v", b)
	}
}

func TestDecodeMap(t *testing.T) {
	var m map[string]int
	if err := Unmarshal([]byte(`a:2:{s:1:"a";i:1;s:1:"b";i:2;}`), &m); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(m, map[string]int{"a": 1, "b": 2}) {
		t.Errorf("got %v", m)
	}
	var mi map[int]string
	if err := Unmarshal([]byte(`a:2:{i:1;s:1:"a";i:2;s:1:"b";}`), &mi); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(mi, map[int]string{1: "a", 2: "b"}) {
		t.Errorf("got %v", mi)
	}
}

func TestDecodeStructFromArray(t *testing.T) {
	type T struct {
		Name string `php:"name"`
		Age  int    `php:"age"`
	}
	var got T
	in := `a:2:{s:4:"name";s:3:"Bob";s:3:"age";i:30;}`
	if err := Unmarshal([]byte(in), &got); err != nil {
		t.Fatal(err)
	}
	if got != (T{Name: "Bob", Age: 30}) {
		t.Errorf("got %+v", got)
	}
}

func TestDecodeStructFromObject(t *testing.T) {
	type T struct {
		Value int `php:"value"`
	}
	var got T
	in := `O:7:"MyClass":1:{s:5:"value";i:42;}`
	if err := Unmarshal([]byte(in), &got); err != nil {
		t.Fatal(err)
	}
	if got.Value != 42 {
		t.Errorf("got %+v", got)
	}
}

func TestDecodeNestedStruct(t *testing.T) {
	type Inner struct {
		X int `php:"x"`
	}
	type Outer struct {
		Inner Inner  `php:"inner"`
		Name  string `php:"name"`
	}
	var got Outer
	in := `a:2:{s:5:"inner";a:1:{s:1:"x";i:5;}s:4:"name";s:1:"y";}`
	if err := Unmarshal([]byte(in), &got); err != nil {
		t.Fatal(err)
	}
	if got.Inner.X != 5 || got.Name != "y" {
		t.Errorf("got %+v", got)
	}
}

func TestDecodePointerFields(t *testing.T) {
	type T struct {
		P *int `php:"p"`
	}
	var got T
	if err := Unmarshal([]byte(`a:1:{s:1:"p";i:7;}`), &got); err != nil {
		t.Fatal(err)
	}
	if got.P == nil || *got.P != 7 {
		t.Errorf("got %+v", got)
	}
	var got2 T
	if err := Unmarshal([]byte(`a:1:{s:1:"p";N;}`), &got2); err != nil {
		t.Fatal(err)
	}
	if got2.P != nil {
		t.Errorf("expected nil pointer, got %v", *got2.P)
	}
}

func TestDecodeUnknownFieldsIgnored(t *testing.T) {
	type T struct {
		A int `php:"a"`
	}
	var got T
	in := `a:2:{s:1:"a";i:1;s:7:"unknown";i:99;}`
	if err := Unmarshal([]byte(in), &got); err != nil {
		t.Fatal(err)
	}
	if got.A != 1 {
		t.Errorf("got %+v", got)
	}
}

func TestDecodeMissingFieldsStayZero(t *testing.T) {
	type T struct {
		A int `php:"a"`
		B int `php:"b"`
	}
	var got T
	if err := Unmarshal([]byte(`a:1:{s:1:"a";i:1;}`), &got); err != nil {
		t.Fatal(err)
	}
	if got != (T{A: 1, B: 0}) {
		t.Errorf("got %+v", got)
	}
}

func TestDecodeCaseInsensitiveFallback(t *testing.T) {
	type T struct {
		Name string `php:"name"`
	}
	var got T
	// PHP key "Name" should match field tagged "name" case-insensitively.
	if err := Unmarshal([]byte(`a:1:{s:4:"Name";s:1:"x";}`), &got); err != nil {
		t.Fatal(err)
	}
	if got.Name != "x" {
		t.Errorf("got %+v", got)
	}
}

func TestDecodeIntoInterface(t *testing.T) {
	tests := []struct {
		in   string
		want any
	}{
		{"N;", nil},
		{"b:1;", true},
		{"i:42;", int64(42)},
		{"d:1.5;", 1.5},
		{`s:2:"hi";`, "hi"},
	}
	for _, tt := range tests {
		var got any
		if err := Unmarshal([]byte(tt.in), &got); err != nil {
			t.Fatalf("Unmarshal(%q): %v", tt.in, err)
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("Unmarshal(%q) = %#v (%T), want %#v", tt.in, got, got, tt.want)
		}
	}
}

func TestDecodeSequentialIntoInterface(t *testing.T) {
	var got any
	if err := Unmarshal([]byte("a:2:{i:0;i:1;i:1;i:2;}"), &got); err != nil {
		t.Fatal(err)
	}
	want := []any{int64(1), int64(2)}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
}

func TestDecodeAssociativeIntoInterface(t *testing.T) {
	var got any
	// Non-sequential integer keys -> map[any]any.
	if err := Unmarshal([]byte("a:2:{i:1;i:10;i:2;i:20;}"), &got); err != nil {
		t.Fatal(err)
	}
	want := map[any]any{int64(1): int64(10), int64(2): int64(20)}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
}

func TestDecodeMixedIntoInterface(t *testing.T) {
	var got any
	if err := Unmarshal([]byte(`a:2:{i:0;s:1:"a";s:1:"k";i:9;}`), &got); err != nil {
		t.Fatal(err)
	}
	want := map[any]any{int64(0): "a", "k": int64(9)}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
}

func TestDecodeObjectIntoInterface(t *testing.T) {
	var got any
	in := `O:4:"User":1:{s:4:"name";s:3:"Bob";}`
	if err := Unmarshal([]byte(in), &got); err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"name": "Bob"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
}

func TestDecodeInvalidDestination(t *testing.T) {
	var iue *InvalidUnmarshalError
	if err := Unmarshal([]byte("N;"), nil); !errors.As(err, &iue) {
		t.Errorf("nil dst: err = %v, want InvalidUnmarshalError", err)
	}
	var i int
	if err := Unmarshal([]byte("N;"), i); !errors.As(err, &iue) {
		t.Errorf("non-pointer dst: err = %v, want InvalidUnmarshalError", err)
	}
	var p *int
	if err := Unmarshal([]byte("N;"), p); !errors.As(err, &iue) {
		t.Errorf("nil pointer dst: err = %v, want InvalidUnmarshalError", err)
	}
}

func TestDecodeInfNaNIntoInterface(t *testing.T) {
	var got any
	if err := Unmarshal([]byte("d:INF;"), &got); err != nil {
		t.Fatal(err)
	}
	if f, ok := got.(float64); !ok || !math.IsInf(f, 1) {
		t.Errorf("got %#v, want +Inf", got)
	}
}

func TestDecodeEmbeddedStruct(t *testing.T) {
	var got WithEmbedded
	in := `a:2:{s:2:"id";i:7;s:4:"name";s:1:"x";}`
	if err := Unmarshal([]byte(in), &got); err != nil {
		t.Fatal(err)
	}
	if got.ID != 7 || got.Name != "x" {
		t.Errorf("got %+v", got)
	}
}
