package phpserialize

import (
	"reflect"
	"testing"
)

// --- Embedded field promotion, dominance and conflicts ---

func TestFieldDominanceShallowWins(t *testing.T) {
	type Inner struct {
		X int `php:"x"`
	}
	type Outer struct {
		X int `php:"x"` // depth 0 dominates Inner.X at depth 1
		Inner
	}
	got := mustMarshal(t, Outer{X: 1, Inner: Inner{X: 2}})
	want := `a:1:{s:1:"x";i:1;}`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFieldConflictDropped(t *testing.T) {
	type A struct {
		X int `php:"x"`
	}
	type B struct {
		X int `php:"x"`
	}
	type C struct {
		A // A.X and B.X both at depth 1 -> ambiguous -> dropped
		B
		Y int `php:"y"`
	}
	got := mustMarshal(t, C{A: A{X: 1}, B: B{X: 2}, Y: 9})
	want := `a:1:{s:1:"y";i:9;}`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestEmbeddedPointer(t *testing.T) {
	type EP struct {
		*Embedded
		Name string `php:"name"`
	}
	// nil embedded pointer: its fields are absent.
	got := mustMarshal(t, EP{Name: "x"})
	want := `a:1:{s:4:"name";s:1:"x";}`
	if got != want {
		t.Errorf("nil embed: got %q, want %q", got, want)
	}
	// populated embedded pointer: its fields are promoted.
	got = mustMarshal(t, EP{Embedded: &Embedded{ID: 3}, Name: "x"})
	want = `a:2:{s:2:"id";i:3;s:4:"name";s:1:"x";}`
	if got != want {
		t.Errorf("populated embed: got %q, want %q", got, want)
	}

	// decode allocates the embedded pointer.
	var dec EP
	if err := Unmarshal([]byte(`a:2:{s:2:"id";i:5;s:4:"name";s:1:"y";}`), &dec); err != nil {
		t.Fatal(err)
	}
	if dec.Embedded == nil || dec.Embedded.ID != 5 || dec.Name != "y" {
		t.Errorf("decoded %+v", dec)
	}
}

type unexpEmbed struct {
	X int `php:"x"`
}

type hasUnexp struct {
	unexpEmbed
	Y int `php:"y"`
}

func TestUnexportedEmbeddedStruct(t *testing.T) {
	// Exported fields of an unexported embedded struct are promoted, both when
	// encoding and decoding.
	got := mustMarshal(t, hasUnexp{unexpEmbed: unexpEmbed{X: 1}, Y: 2})
	want := `a:2:{s:1:"x";i:1;s:1:"y";i:2;}`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	var out hasUnexp
	if err := Unmarshal([]byte(want), &out); err != nil {
		t.Fatal(err)
	}
	if out.X != 1 || out.Y != 2 {
		t.Errorf("decoded %+v", out)
	}
}

// --- Map key encoding ---

func TestEncodeMapKeyKinds(t *testing.T) {
	if got := mustMarshal(t, map[int8]int{1: 10}); got != `a:1:{i:1;i:10;}` {
		t.Errorf("int8 key: %q", got)
	}
	if got := mustMarshal(t, map[uint16]int{2: 20}); got != `a:1:{i:2;i:20;}` {
		t.Errorf("uint16 key: %q", got)
	}
}

func TestEncodeMapAnyKey(t *testing.T) {
	// map[any]any with mixed keys round-trips through the interface path.
	m := map[any]any{"s": int64(1), int64(2): "v"}
	b, err := Marshal(m)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var out any
	if err := Unmarshal(b, &out); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	got, ok := out.(map[any]any)
	if !ok {
		t.Fatalf("got %T", out)
	}
	if !reflect.DeepEqual(got, m) {
		t.Errorf("got %#v, want %#v", got, m)
	}
}

func TestEncodeMapUnsupportedKey(t *testing.T) {
	m := map[float64]int{1.5: 1}
	if _, err := Marshal(m); err == nil {
		t.Error("expected error for float map key")
	}
}

// --- isEmptyValue coverage across kinds ---

func TestOmitEmptyKinds(t *testing.T) {
	type T struct {
		Sl  []int          `php:"sl,omitempty"`
		Mp  map[string]int `php:"mp,omitempty"`
		Ptr *int           `php:"ptr,omitempty"`
		F   float64        `php:"f,omitempty"`
		U   uint           `php:"u,omitempty"`
		B   bool           `php:"b,omitempty"`
		Ar  [0]int         `php:"ar,omitempty"`
	}
	got := mustMarshal(t, T{})
	if got != "a:0:{}" {
		t.Errorf("all-empty struct = %q, want a:0:{}", got)
	}
	n := 1
	got = mustMarshal(t, T{Sl: []int{1}, Mp: map[string]int{"a": 1}, Ptr: &n, F: 1, U: 1, B: true})
	// Ar is a zero-length array, always empty, so it is omitted.
	if got == "a:0:{}" {
		t.Error("expected populated fields to be present")
	}
}

// --- Float -> unsigned integer decode ---

func TestDecodeFloatIntoUint(t *testing.T) {
	var u uint
	if err := Unmarshal([]byte("d:5;"), &u); err != nil || u != 5 {
		t.Errorf("float->uint: %v %v", u, err)
	}
	var u2 uint
	if err := Unmarshal([]byte("d:-1;"), &u2); err == nil {
		t.Error("negative float into uint should fail")
	}
	var u3 uint
	if err := Unmarshal([]byte("d:1.5;"), &u3); err == nil {
		t.Error("non-integral float into uint should fail")
	}
}

// --- Decode into map with interface keys/values ---

func TestDecodeIntoConcreteAnyMap(t *testing.T) {
	var m map[any]any
	if err := Unmarshal([]byte(`a:2:{i:5;s:1:"a";s:1:"k";i:9;}`), &m); err != nil {
		t.Fatal(err)
	}
	want := map[any]any{int64(5): "a", "k": int64(9)}
	if !reflect.DeepEqual(m, want) {
		t.Errorf("got %#v, want %#v", m, want)
	}
}

// --- Object decoded into a map ---

func TestDecodeObjectIntoMap(t *testing.T) {
	var m map[string]int
	if err := Unmarshal([]byte(`O:1:"A":2:{s:1:"a";i:1;s:1:"b";i:2;}`), &m); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(m, map[string]int{"a": 1, "b": 2}) {
		t.Errorf("got %#v", m)
	}
}

// --- Error paths: wrong destinations for composite PHP values ---

func TestDecodeCompositeTypeMismatches(t *testing.T) {
	var i int
	if err := Unmarshal([]byte("a:0:{}"), &i); err == nil {
		t.Error("array into int should fail")
	}
	var s string
	if err := Unmarshal([]byte(`O:1:"A":0:{}`), &s); err == nil {
		t.Error("object into string should fail")
	}
	// bad map key type: string PHP key into int-keyed map.
	var m map[int]int
	if err := Unmarshal([]byte(`a:1:{s:1:"x";i:1;}`), &m); err == nil {
		t.Error("string key into map[int]int should fail")
	}
	// element decode error propagates.
	var sl []int
	if err := Unmarshal([]byte(`a:1:{i:0;s:1:"x";}`), &sl); err == nil {
		t.Error("string element into []int should fail")
	}
}

// --- Non-empty interface destination is rejected ---

type notImplemented interface{ Foo() }

func TestDecodeNonEmptyInterface(t *testing.T) {
	var x notImplemented
	if err := Unmarshal([]byte("i:1;"), &x); err == nil {
		t.Error("decoding into non-empty interface should fail")
	}
}

func TestUnmarshalParseErrorPropagates(t *testing.T) {
	var v any
	if err := Unmarshal([]byte("i:not-a-number;"), &v); err == nil {
		t.Error("expected parse error to propagate through Unmarshal")
	}
}

func TestDecodeBoolMismatch(t *testing.T) {
	var s string
	if err := Unmarshal([]byte("b:1;"), &s); err == nil {
		t.Error("bool into string should fail")
	}
}

// --- PHPClass on a non-struct kind ---

type IntClass int

func (IntClass) PHPClassName() string { return "IntBox" }

func TestEncodePHPClassNonStruct(t *testing.T) {
	// A PHPClass whose underlying kind is not a struct yields an object with no
	// fields but the requested class name.
	got := mustMarshal(t, IntClass(5))
	want := `O:6:"IntBox":0:{}`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// --- Encode nil interface inside a slice ---

func TestEncodeNilPHPClassPointer(t *testing.T) {
	var p *PtrClass
	if got := mustMarshal(t, p); got != "N;" {
		t.Errorf("nil PHPClass pointer = %q, want N;", got)
	}
}
