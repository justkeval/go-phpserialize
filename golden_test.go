package phpserialize

import "testing"

// goldenCases are canonical PHP serialized strings as produced by PHP's own
// serialize() function (serialize_precision = -1). Each is parsed and then
// re-written; the re-written bytes must match the input exactly, exercising
// both the parser and the writer against authoritative output.
var goldenCases = []struct {
	name string
	data string
}{
	{"null", "N;"},
	{"true", "b:1;"},
	{"false", "b:0;"},
	{"int zero", "i:0;"},
	{"int", "i:42;"},
	{"int negative", "i:-1;"},
	{"int large", "i:9223372036854775807;"},
	{"float", "d:1.5;"},
	{"float integral", "d:3;"},
	{"float exp", "d:1.0E+20;"},
	{"string", `s:5:"Hello";`},
	{"string empty", `s:0:"";`},
	{"string unicode", `s:6:"héllo";`},
	{"list", "a:3:{i:0;i:1;i:1;i:2;i:2;i:3;}"},
	{"assoc", `a:2:{s:1:"a";i:1;s:1:"b";i:2;}`},
	{"nested assoc", `a:1:{s:1:"x";a:1:{s:1:"y";s:1:"z";}}`},
	{"mixed nested", `a:2:{s:2:"id";i:1;s:4:"tags";a:2:{i:0;s:3:"php";i:1;s:2:"go";}}`},
	{"empty array", "a:0:{}"},
	{"stdClass", `O:8:"stdClass":1:{s:4:"name";s:3:"Bob";}`},
	{"typed class", `O:5:"Point":2:{s:1:"x";i:1;s:1:"y";i:2;}`},
	{"object with nested array", `O:4:"Data":1:{s:5:"items";a:2:{i:0;i:10;i:1;i:20;}}`},
}

func TestGoldenParseAndWrite(t *testing.T) {
	for _, tc := range goldenCases {
		t.Run(tc.name, func(t *testing.T) {
			v, err := Parse([]byte(tc.data))
			if err != nil {
				t.Fatalf("Parse(%q): %v", tc.data, err)
			}
			out, err := Write(v)
			if err != nil {
				t.Fatalf("Write: %v", err)
			}
			if string(out) != tc.data {
				t.Errorf("round-trip mismatch:\n got  %q\n want %q", out, tc.data)
			}
		})
	}
}

// TestGoldenIntoInterface decodes selected golden values into an empty
// interface and checks the resulting native Go shapes.
func TestGoldenIntoInterface(t *testing.T) {
	var out any
	if err := Unmarshal([]byte(`O:8:"stdClass":2:{s:4:"name";s:3:"Bob";s:3:"age";i:30;}`), &out); err != nil {
		t.Fatal(err)
	}
	m, ok := out.(map[string]any)
	if !ok {
		t.Fatalf("got %T, want map[string]any", out)
	}
	if m["name"] != "Bob" || m["age"] != int64(30) {
		t.Errorf("got %#v", m)
	}
}
