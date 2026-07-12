package phpserialize

import (
	"math"
	"testing"
)

func TestWriteScalars(t *testing.T) {
	tests := []struct {
		name string
		val  Value
		want string
	}{
		{"null", Null(), "N;"},
		{"true", Bool(true), "b:1;"},
		{"false", Bool(false), "b:0;"},
		{"int zero", Int(0), "i:0;"},
		{"int positive", Int(42), "i:42;"},
		{"int negative", Int(-7), "i:-7;"},
		{"int max", Int(math.MaxInt64), "i:9223372036854775807;"},
		{"int min", Int(math.MinInt64), "i:-9223372036854775808;"},
		{"string", String("hello"), `s:5:"hello";`},
		{"string empty", String(""), `s:0:"";`},
		{"string unicode", String("héllo"), `s:6:"héllo";`}, // é is 2 bytes
		{"string emoji", String("😀"), `s:4:"😀";`},           // 4 bytes
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Write(tt.val)
			if err != nil {
				t.Fatalf("Write: %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("Write() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestWriteFloat(t *testing.T) {
	// Expected canonical PHP serialize() output (serialize_precision = -1).
	tests := []struct {
		in   float64
		want string
	}{
		{0, "d:0;"},
		{1, "d:1;"},
		{1.5, "d:1.5;"},
		{-1.5, "d:-1.5;"},
		{0.1, "d:0.1;"},
		{0.0001, "d:0.0001;"},
		{0.00001, "d:1.0E-5;"},
		{100, "d:100;"},
		{3.14, "d:3.14;"},
		{1e20, "d:1.0E+20;"},
		{1e-20, "d:1.0E-20;"},
		{1e15, "d:1000000000000000;"},
		{1e16, "d:10000000000000000;"},
		{1e17, "d:1.0E+17;"},
		{123456789, "d:123456789;"},
		{0.5, "d:0.5;"},
		{2.5, "d:2.5;"},
		{1234567890123456789.0, "d:1.2345678901234568E+18;"},
		{math.Inf(1), "d:INF;"},
		{math.Inf(-1), "d:-INF;"},
		{math.NaN(), "d:NAN;"},
		{math.Copysign(0, -1), "d:-0;"},
	}
	for _, tt := range tests {
		got, err := Write(Float(tt.in))
		if err != nil {
			t.Fatalf("Write(%v): %v", tt.in, err)
		}
		if string(got) != tt.want {
			t.Errorf("Write(Float(%v)) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestWriteArray(t *testing.T) {
	// Sequential array [10, 20].
	seq := Value{Kind: ArrayKind, Array: []Entry{
		{Key: Int(0), Value: Int(10)},
		{Key: Int(1), Value: Int(20)},
	}}
	got, err := Write(seq)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	want := "a:2:{i:0;i:10;i:1;i:20;}"
	if string(got) != want {
		t.Errorf("Write() = %q, want %q", got, want)
	}

	// Associative array {"a": 1, "b": "x"}.
	assoc := Value{Kind: ArrayKind, Array: []Entry{
		{Key: String("a"), Value: Int(1)},
		{Key: String("b"), Value: String("x")},
	}}
	got, err = Write(assoc)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	want = `a:2:{s:1:"a";i:1;s:1:"b";s:1:"x";}`
	if string(got) != want {
		t.Errorf("Write() = %q, want %q", got, want)
	}
}

func TestWriteNestedArray(t *testing.T) {
	nested := Value{Kind: ArrayKind, Array: []Entry{
		{Key: String("list"), Value: Value{Kind: ArrayKind, Array: []Entry{
			{Key: Int(0), Value: Int(1)},
			{Key: Int(1), Value: Int(2)},
		}}},
	}}
	got, err := Write(nested)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	want := `a:1:{s:4:"list";a:2:{i:0;i:1;i:1;i:2;}}`
	if string(got) != want {
		t.Errorf("Write() = %q, want %q", got, want)
	}
}

func TestWriteObject(t *testing.T) {
	obj := Value{Kind: ObjectKind, Object: &Object{
		ClassName: "User",
		Fields: []Field{
			{Name: "name", Value: String("Bob")},
			{Name: "age", Value: Int(30)},
		},
	}}
	got, err := Write(obj)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	want := `O:4:"User":2:{s:4:"name";s:3:"Bob";s:3:"age";i:30;}`
	if string(got) != want {
		t.Errorf("Write() = %q, want %q", got, want)
	}
}

func TestWriteInvalidArrayKey(t *testing.T) {
	bad := Value{Kind: ArrayKind, Array: []Entry{
		{Key: Bool(true), Value: Int(1)},
	}}
	if _, err := Write(bad); err == nil {
		t.Fatal("expected error for invalid array key kind")
	}
}

func TestWriteUnknownKind(t *testing.T) {
	if _, err := Write(Value{Kind: Kind(99)}); err == nil {
		t.Fatal("expected error for unknown kind")
	}
}
