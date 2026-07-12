package phpserialize

import "testing"

func TestParseTag(t *testing.T) {
	tests := []struct {
		in   string
		want tagOptions
	}{
		{"", tagOptions{name: ""}},
		{"name", tagOptions{name: "name"}},
		{"name,omitempty", tagOptions{name: "name", omitEmpty: true}},
		{",omitempty", tagOptions{name: "", omitEmpty: true}},
		{"-", tagOptions{skip: true}},
		{"-,", tagOptions{name: "-"}}, // "-," names the field "-"
		{"name,unknown", tagOptions{name: "name"}},
		{"name,omitempty,extra", tagOptions{name: "name", omitEmpty: true}},
	}
	for _, tt := range tests {
		got := parseTag(tt.in)
		if got != tt.want {
			t.Errorf("parseTag(%q) = %+v, want %+v", tt.in, got, tt.want)
		}
	}
}

func TestParseTagDashNamedField(t *testing.T) {
	// A field tagged `php:"-,"` should be encoded under the literal name "-".
	type T struct {
		Keep int `php:"-,"`
		Skip int `php:"-"`
	}
	got, err := Marshal(T{Keep: 1, Skip: 2})
	if err != nil {
		t.Fatal(err)
	}
	want := `a:1:{s:1:"-";i:1;}`
	if string(got) != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestIsValidTagName(t *testing.T) {
	if isValidTagName("") {
		t.Error("empty name should be invalid")
	}
	if !isValidTagName("field") {
		t.Error("non-empty name should be valid")
	}
}
