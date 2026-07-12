package phpserialize

import (
	"errors"
	"math"
	"testing"
)

func TestParseScalars(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want Value
	}{
		{"null", "N;", Null()},
		{"true", "b:1;", Bool(true)},
		{"false", "b:0;", Bool(false)},
		{"int", "i:42;", Int(42)},
		{"int negative", "i:-7;", Int(-7)},
		{"int zero", "i:0;", Int(0)},
		{"int max", "i:9223372036854775807;", Int(math.MaxInt64)},
		{"int min", "i:-9223372036854775808;", Int(math.MinInt64)},
		{"float", "d:1.5;", Float(1.5)},
		{"float negative", "d:-2.25;", Float(-2.25)},
		{"float exp", "d:1.0E+20;", Float(1e20)},
		{"float small", "d:1.0E-5;", Float(0.00001)},
		{"float int-like", "d:100;", Float(100)},
		{"string", `s:5:"hello";`, String("hello")},
		{"string empty", `s:0:"";`, String("")},
		{"string unicode", `s:6:"héllo";`, String("héllo")},
		{"string with quotes", `s:5:"a"b"c";`, String(`a"b"c`)},
		{"string with braces", `s:3:"{};";`, String("{};")},
		{"string with semicolons", `s:3:"a;b";`, String("a;b")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse([]byte(tt.in))
			if err != nil {
				t.Fatalf("Parse(%q): %v", tt.in, err)
			}
			if !got.Equal(tt.want) {
				t.Errorf("Parse(%q) = %+v, want %+v", tt.in, got, tt.want)
			}
		})
	}
}

func TestParseFloatSpecials(t *testing.T) {
	tests := []struct {
		in   string
		want float64
	}{
		{"d:INF;", math.Inf(1)},
		{"d:-INF;", math.Inf(-1)},
	}
	for _, tt := range tests {
		got, err := Parse([]byte(tt.in))
		if err != nil {
			t.Fatalf("Parse(%q): %v", tt.in, err)
		}
		if got.Kind != FloatKind || got.Float != tt.want {
			t.Errorf("Parse(%q) = %+v, want float %v", tt.in, got, tt.want)
		}
	}
	got, err := Parse([]byte("d:NAN;"))
	if err != nil {
		t.Fatalf("Parse NAN: %v", err)
	}
	if got.Kind != FloatKind || !math.IsNaN(got.Float) {
		t.Errorf("Parse NAN = %+v, want NaN", got)
	}
}

func TestParseSequentialArray(t *testing.T) {
	got, err := Parse([]byte("a:3:{i:0;i:10;i:1;i:20;i:2;i:30;}"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	want := Value{Kind: ArrayKind, Array: []Entry{
		{Key: Int(0), Value: Int(10)},
		{Key: Int(1), Value: Int(20)},
		{Key: Int(2), Value: Int(30)},
	}}
	if !got.Equal(want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestParseAssociativeArray(t *testing.T) {
	got, err := Parse([]byte(`a:2:{s:4:"name";s:3:"Bob";s:3:"age";i:30;}`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	want := Value{Kind: ArrayKind, Array: []Entry{
		{Key: String("name"), Value: String("Bob")},
		{Key: String("age"), Value: Int(30)},
	}}
	if !got.Equal(want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestParseMixedArray(t *testing.T) {
	got, err := Parse([]byte(`a:2:{i:0;s:1:"a";s:3:"key";i:99;}`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	want := Value{Kind: ArrayKind, Array: []Entry{
		{Key: Int(0), Value: String("a")},
		{Key: String("key"), Value: Int(99)},
	}}
	if !got.Equal(want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestParseNestedArray(t *testing.T) {
	got, err := Parse([]byte("a:1:{i:0;a:2:{i:0;i:1;i:1;i:2;}}"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	inner := Value{Kind: ArrayKind, Array: []Entry{
		{Key: Int(0), Value: Int(1)},
		{Key: Int(1), Value: Int(2)},
	}}
	want := Value{Kind: ArrayKind, Array: []Entry{{Key: Int(0), Value: inner}}}
	if !got.Equal(want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestParseEmptyArray(t *testing.T) {
	got, err := Parse([]byte("a:0:{}"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got.Kind != ArrayKind || len(got.Array) != 0 {
		t.Errorf("got %+v, want empty array", got)
	}
}

func TestParseObject(t *testing.T) {
	got, err := Parse([]byte(`O:4:"User":2:{s:4:"name";s:3:"Bob";s:3:"age";i:30;}`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got.Kind != ObjectKind {
		t.Fatalf("kind = %v, want object", got.Kind)
	}
	if got.Object.ClassName != "User" {
		t.Errorf("class = %q, want User", got.Object.ClassName)
	}
	want := Value{Kind: ObjectKind, Object: &Object{
		ClassName: "User",
		Fields: []Field{
			{Name: "name", Value: String("Bob")},
			{Name: "age", Value: Int(30)},
		},
	}}
	if !got.Equal(want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestParseNestedObject(t *testing.T) {
	in := `O:1:"A":1:{s:1:"b";O:1:"B":1:{s:1:"c";i:1;}}`
	got, err := Parse([]byte(in))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got.Object.Fields[0].Value.Object.ClassName != "B" {
		t.Errorf("nested class = %q, want B", got.Object.Fields[0].Value.Object.ClassName)
	}
}

func TestParseLargeAndNegative(t *testing.T) {
	got, err := Parse([]byte("i:-9223372036854775808;"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got.Int != math.MinInt64 {
		t.Errorf("got %d, want MinInt64", got.Int)
	}
}

func TestParseUnsupportedTypes(t *testing.T) {
	for _, in := range []string{
		`C:3:"Foo":3:{bar}`,
		"R:1;",
		"r:1;",
	} {
		_, err := Parse([]byte(in))
		if !errors.Is(err, ErrUnsupportedType) {
			t.Errorf("Parse(%q) error = %v, want ErrUnsupportedType", in, err)
		}
	}
}

func TestParseMalformed(t *testing.T) {
	tests := []struct {
		name string
		in   string
	}{
		{"empty", ""},
		{"null no semicolon", "N"},
		{"bool bad value", "b:2;"},
		{"bool no semicolon", "b:1"},
		{"int no colon", "i42;"},
		{"int malformed", "i:12x;"},
		{"int empty", "i:;"},
		{"int overflow", "i:99999999999999999999;"},
		{"float malformed", "d:1.2.3;"},
		{"string bad length", `s:5:"abc";`},
		{"string missing open quote", `s:3:abc";`},
		{"string missing close quote", `s:3:"abc;`},
		{"string length exceeds input", `s:100:"abc";`},
		{"string negative length", `s:-1:"";`},
		{"array bad count", "a:2:{i:0;i:1;}"},
		{"array no open brace", "a:0:}"},
		{"array unclosed", "a:1:{i:0;i:1;"},
		{"array missing separator", "a1:{}"},
		{"object bad name len", `O:9:"User":0:{}`},
		{"object unclosed", `O:4:"User":1:{s:1:"a";i:1;`},
		{"object non-string key", `O:1:"A":1:{i:0;i:1;}`},
		{"trailing data", "N;N;"},
		{"unknown marker", "X:1;"},
		{"array non-scalar key", "a:1:{a:0:{}i:1;}"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse([]byte(tt.in))
			if err == nil {
				t.Errorf("Parse(%q) succeeded, want error", tt.in)
			}
		})
	}
}

func TestParseSyntaxErrorOffset(t *testing.T) {
	_, err := Parse([]byte("b:2;"))
	var se *SyntaxError
	if !errors.As(err, &se) {
		t.Fatalf("error = %v, want *SyntaxError", err)
	}
	if se.Offset != 2 {
		t.Errorf("offset = %d, want 2", se.Offset)
	}
}
