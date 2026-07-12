package phpserialize

import (
	"math"
	"testing"
)

func TestKindString(t *testing.T) {
	tests := []struct {
		k    Kind
		want string
	}{
		{NullKind, "null"},
		{BoolKind, "bool"},
		{IntKind, "int"},
		{FloatKind, "float"},
		{StringKind, "string"},
		{ArrayKind, "array"},
		{ObjectKind, "object"},
		{Kind(200), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.k.String(); got != tt.want {
			t.Errorf("Kind(%d).String() = %q, want %q", tt.k, got, tt.want)
		}
	}
}

func TestValueConstructors(t *testing.T) {
	if Null().Kind != NullKind {
		t.Error("Null")
	}
	if v := Bool(true); v.Kind != BoolKind || !v.Bool {
		t.Error("Bool")
	}
	if v := Int(5); v.Kind != IntKind || v.Int != 5 {
		t.Error("Int")
	}
	if v := Float(1.5); v.Kind != FloatKind || v.Float != 1.5 {
		t.Error("Float")
	}
	if v := String("x"); v.Kind != StringKind || v.String != "x" {
		t.Error("String")
	}
}

func TestValueEqual(t *testing.T) {
	if !Int(1).Equal(Int(1)) {
		t.Error("equal ints")
	}
	if Int(1).Equal(Int(2)) {
		t.Error("unequal ints")
	}
	if Int(1).Equal(Float(1)) {
		t.Error("different kinds")
	}
	// NaN equals NaN under our definition (for stable round-trip tests).
	if !Float(math.NaN()).Equal(Float(math.NaN())) {
		t.Error("NaN should equal NaN")
	}
	// arrays differ by length
	a := Value{Kind: ArrayKind, Array: []Entry{{Key: Int(0), Value: Int(1)}}}
	b := Value{Kind: ArrayKind}
	if a.Equal(b) {
		t.Error("arrays of different length should not be equal")
	}
	// objects: nil vs non-nil
	o1 := Value{Kind: ObjectKind, Object: &Object{ClassName: "A"}}
	o2 := Value{Kind: ObjectKind}
	if o1.Equal(o2) {
		t.Error("object nil mismatch")
	}
	// objects differ by class name
	o3 := Value{Kind: ObjectKind, Object: &Object{ClassName: "B"}}
	if o1.Equal(o3) {
		t.Error("objects with different class names should differ")
	}
	// unknown kinds are never equal
	u := Value{Kind: Kind(50)}
	if u.Equal(u) {
		t.Error("unknown kinds should not compare equal")
	}
}
