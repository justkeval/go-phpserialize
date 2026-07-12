package phpserialize

import (
	"reflect"
	"testing"
)

// TestValueRoundTrip verifies Value -> Write -> Parse -> Value is the identity
// for representative values.
func TestValueRoundTrip(t *testing.T) {
	values := []Value{
		Null(),
		Bool(true),
		Bool(false),
		Int(0),
		Int(-123456789),
		Float(3.14159),
		Float(1e20),
		String(""),
		String("hello world"),
		String("unicode: héllo 😀"),
		{Kind: ArrayKind, Array: []Entry{
			{Key: Int(0), Value: Int(1)},
			{Key: Int(1), Value: String("two")},
			{Key: String("k"), Value: Bool(true)},
		}},
		{Kind: ObjectKind, Object: &Object{
			ClassName: "Widget",
			Fields: []Field{
				{Name: "id", Value: Int(1)},
				{Name: "tags", Value: Value{Kind: ArrayKind, Array: []Entry{
					{Key: Int(0), Value: String("a")},
				}}},
			},
		}},
	}
	for i, v := range values {
		b, err := Write(v)
		if err != nil {
			t.Fatalf("[%d] Write: %v", i, err)
		}
		got, err := Parse(b)
		if err != nil {
			t.Fatalf("[%d] Parse(%q): %v", i, b, err)
		}
		if !got.Equal(v) {
			t.Errorf("[%d] round-trip mismatch:\n got  %+v\n want %+v", i, got, v)
		}
	}
}

// TestGoRoundTrip verifies Go -> Marshal -> Unmarshal -> Go for representative
// Go values.
func TestGoRoundTrip(t *testing.T) {
	t.Run("scalars", func(t *testing.T) {
		type S struct {
			B  bool    `php:"b"`
			I  int     `php:"i"`
			U  uint    `php:"u"`
			F  float64 `php:"f"`
			St string  `php:"s"`
		}
		in := S{B: true, I: -42, U: 99, F: 2.5, St: "hi"}
		var out S
		roundTripGo(t, in, &out)
		if in != out {
			t.Errorf("got %+v, want %+v", out, in)
		}
	})

	t.Run("slice", func(t *testing.T) {
		in := []string{"a", "b", "c"}
		var out []string
		roundTripGo(t, in, &out)
		if !reflect.DeepEqual(in, out) {
			t.Errorf("got %v, want %v", out, in)
		}
	})

	t.Run("map", func(t *testing.T) {
		in := map[string]int{"x": 1, "y": 2}
		var out map[string]int
		roundTripGo(t, in, &out)
		if !reflect.DeepEqual(in, out) {
			t.Errorf("got %v, want %v", out, in)
		}
	})

	t.Run("nested", func(t *testing.T) {
		type Inner struct {
			Vals []int `php:"vals"`
		}
		type Outer struct {
			Name  string          `php:"name"`
			Inner Inner           `php:"inner"`
			Meta  map[string]bool `php:"meta"`
			Ptr   *int            `php:"ptr"`
		}
		n := 7
		in := Outer{
			Name:  "test",
			Inner: Inner{Vals: []int{1, 2, 3}},
			Meta:  map[string]bool{"ok": true},
			Ptr:   &n,
		}
		var out Outer
		roundTripGo(t, in, &out)
		if !reflect.DeepEqual(in, out) {
			t.Errorf("got %+v, want %+v", out, in)
		}
	})

	t.Run("phpclass", func(t *testing.T) {
		in := MyClass{Value: 123}
		var out MyClass
		roundTripGo(t, in, &out)
		if in != out {
			t.Errorf("got %+v, want %+v", out, in)
		}
	})
}

func roundTripGo(t *testing.T, in any, out any) {
	t.Helper()
	b, err := Marshal(in)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if err := Unmarshal(b, out); err != nil {
		t.Fatalf("Unmarshal(%q): %v", b, err)
	}
}
