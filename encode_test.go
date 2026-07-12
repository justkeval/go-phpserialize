package phpserialize

import (
	"testing"
)

func mustMarshal(t *testing.T, v any) string {
	t.Helper()
	b, err := Marshal(v)
	if err != nil {
		t.Fatalf("Marshal(%#v): %v", v, err)
	}
	return string(b)
}

func TestEncodePrimitives(t *testing.T) {
	tests := []struct {
		name string
		in   any
		want string
	}{
		{"nil", nil, "N;"},
		{"true", true, "b:1;"},
		{"false", false, "b:0;"},
		{"int", 42, "i:42;"},
		{"int8", int8(-5), "i:-5;"},
		{"int16", int16(300), "i:300;"},
		{"int32", int32(-70000), "i:-70000;"},
		{"int64", int64(1 << 40), "i:1099511627776;"},
		{"uint", uint(7), "i:7;"},
		{"uint8", uint8(255), "i:255;"},
		{"uint16", uint16(65535), "i:65535;"},
		{"uint32", uint32(4000000000), "i:4000000000;"},
		{"uint64", uint64(1 << 40), "i:1099511627776;"},
		{"float32", float32(1.5), "d:1.5;"},
		{"float64", 3.14, "d:3.14;"},
		{"string", "hi", `s:2:"hi";`},
		{"string empty", "", `s:0:"";`},
		{"bytes", []byte("abc"), `s:3:"abc";`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mustMarshal(t, tt.in); got != tt.want {
				t.Errorf("Marshal(%#v) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestEncodeSlice(t *testing.T) {
	got := mustMarshal(t, []int{1, 2, 3})
	want := "a:3:{i:0;i:1;i:1;i:2;i:2;i:3;}"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestEncodeArray(t *testing.T) {
	got := mustMarshal(t, [2]string{"a", "b"})
	want := `a:2:{i:0;s:1:"a";i:1;s:1:"b";}`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestEncodeNilSlice(t *testing.T) {
	var s []int
	if got := mustMarshal(t, s); got != "N;" {
		t.Errorf("nil slice = %q, want N;", got)
	}
	if got := mustMarshal(t, []int{}); got != "a:0:{}" {
		t.Errorf("empty slice = %q, want a:0:{}", got)
	}
}

func TestEncodeMapDeterministic(t *testing.T) {
	// String keys are emitted in lexical order.
	got := mustMarshal(t, map[string]int{"b": 2, "a": 1, "c": 3})
	want := `a:3:{s:1:"a";i:1;s:1:"b";i:2;s:1:"c";i:3;}`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	// Integer keys are emitted in numeric order.
	got = mustMarshal(t, map[int]string{3: "c", 1: "a", 2: "b"})
	want = `a:3:{i:1;s:1:"a";i:2;s:1:"b";i:3;s:1:"c";}`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestEncodeNilMap(t *testing.T) {
	var m map[string]int
	if got := mustMarshal(t, m); got != "N;" {
		t.Errorf("nil map = %q, want N;", got)
	}
}

type Address struct {
	City string `php:"city"`
	Zip  string `php:"zip"`
}

type Person struct {
	Name    string  `php:"name"`
	Age     int     `php:"age"`
	Addr    Address `php:"address"`
	private int
}

func TestEncodeStruct(t *testing.T) {
	p := Person{Name: "Bob", Age: 30, Addr: Address{City: "NYC", Zip: "10001"}, private: 5}
	got := mustMarshal(t, p)
	want := `a:3:{s:4:"name";s:3:"Bob";s:3:"age";i:30;s:7:"address";a:2:{s:4:"city";s:3:"NYC";s:3:"zip";s:5:"10001";}}`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestEncodeStructNoTags(t *testing.T) {
	type T struct {
		Foo int
		Bar string
	}
	got := mustMarshal(t, T{Foo: 1, Bar: "x"})
	want := `a:2:{s:3:"Foo";i:1;s:3:"Bar";s:1:"x";}`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestEncodeOmitEmpty(t *testing.T) {
	type T struct {
		A int    `php:"a,omitempty"`
		B string `php:"b,omitempty"`
		C int    `php:"c"`
	}
	got := mustMarshal(t, T{A: 0, B: "", C: 0})
	want := `a:1:{s:1:"c";i:0;}`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	got = mustMarshal(t, T{A: 1, B: "x", C: 2})
	want = `a:3:{s:1:"a";i:1;s:1:"b";s:1:"x";s:1:"c";i:2;}`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestEncodeSkipField(t *testing.T) {
	type T struct {
		A int `php:"a"`
		B int `php:"-"`
	}
	got := mustMarshal(t, T{A: 1, B: 2})
	want := `a:1:{s:1:"a";i:1;}`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestEncodePointers(t *testing.T) {
	n := 5
	if got := mustMarshal(t, &n); got != "i:5;" {
		t.Errorf("*int = %q, want i:5;", got)
	}
	var np *int
	if got := mustMarshal(t, np); got != "N;" {
		t.Errorf("nil *int = %q, want N;", got)
	}
	pp := &n
	if got := mustMarshal(t, &pp); got != "i:5;" {
		t.Errorf("**int = %q, want i:5;", got)
	}
}

func TestEncodeInterface(t *testing.T) {
	var i any = "hello"
	if got := mustMarshal(t, i); got != `s:5:"hello";` {
		t.Errorf("got %q", got)
	}
	vals := []any{1, "two", true, nil}
	got := mustMarshal(t, vals)
	want := `a:4:{i:0;i:1;i:1;s:3:"two";i:2;b:1;i:3;N;}`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

type Embedded struct {
	ID int `php:"id"`
}

type WithEmbedded struct {
	Embedded
	Name string `php:"name"`
}

func TestEncodeEmbedded(t *testing.T) {
	got := mustMarshal(t, WithEmbedded{Embedded: Embedded{ID: 7}, Name: "x"})
	// Promoted embedded fields appear alongside the outer fields.
	want := `a:2:{s:2:"id";i:7;s:4:"name";s:1:"x";}`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

type NamedEmbedded struct {
	Embedded `php:"embedded"`
	Name     string `php:"name"`
}

func TestEncodeNamedEmbedded(t *testing.T) {
	got := mustMarshal(t, NamedEmbedded{Embedded: Embedded{ID: 7}, Name: "x"})
	// A tag on an embedded field turns it into a nested value.
	want := `a:2:{s:8:"embedded";a:1:{s:2:"id";i:7;}s:4:"name";s:1:"x";}`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

type MyClass struct {
	Value int `php:"value"`
}

func (MyClass) PHPClassName() string { return "MyClass" }

func TestEncodePHPClass(t *testing.T) {
	got := mustMarshal(t, MyClass{Value: 42})
	want := `O:7:"MyClass":1:{s:5:"value";i:42;}`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

type PtrClass struct {
	Value int `php:"value"`
}

func (*PtrClass) PHPClassName() string { return "PtrClass" }

func TestEncodePHPClassPointer(t *testing.T) {
	got := mustMarshal(t, &PtrClass{Value: 1})
	want := `O:8:"PtrClass":1:{s:5:"value";i:1;}`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestEncodeUnsupportedType(t *testing.T) {
	ch := make(chan int)
	if _, err := Marshal(ch); err == nil {
		t.Fatal("expected error marshaling a channel")
	}
	fn := func() {}
	if _, err := Marshal(fn); err == nil {
		t.Fatal("expected error marshaling a func")
	}
}

func TestEncodeReturnsValue(t *testing.T) {
	v, err := Encode(map[string]int{"a": 1})
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if v.Kind != ArrayKind || len(v.Array) != 1 {
		t.Errorf("got %+v", v)
	}
}
