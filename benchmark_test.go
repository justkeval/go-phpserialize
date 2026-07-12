package phpserialize

import (
	"encoding/json"
	"testing"
)

type benchUser struct {
	ID       int
	Name     string
	Email    string
	Active   bool
	Score    float64
	Tags     []string
	Metadata map[string]any
}

var (
	benchStruct = benchUser{
		ID:     42,
		Name:   "Alice",
		Email:  "alice@example.com",
		Active: true,
		Score:  98.5,
		Tags:   []string{"go", "php", "serialize"},
		Metadata: map[string]any{
			"country": "India",
			"age":     24,
			"premium": true,
		},
	}

	phpBytes, _  = Marshal(benchStruct)
	jsonBytes, _ = json.Marshal(benchStruct)

	sinkBytes []byte
	sinkUser  benchUser
	sinkValue Value
)

func BenchmarkMarshal(b *testing.B) {
	b.Run("PHPSerialize", func(b *testing.B) {
		b.ReportAllocs()

		for b.Loop() {
			sinkBytes, _ = Marshal(benchStruct)
		}
	})

	b.Run("EncodingJSON", func(b *testing.B) {
		b.ReportAllocs()

		for b.Loop() {
			sinkBytes, _ = json.Marshal(benchStruct)
		}
	})
}

func BenchmarkUnmarshal(b *testing.B) {
	b.Run("PHPSerialize", func(b *testing.B) {
		b.ReportAllocs()

		for b.Loop() {
			var v benchUser
			_ = Unmarshal(phpBytes, &v)
			sinkUser = v
		}
	})

	b.Run("EncodingJSON", func(b *testing.B) {
		b.ReportAllocs()

		for b.Loop() {
			var v benchUser
			_ = json.Unmarshal(jsonBytes, &v)
			sinkUser = v
		}
	})
}

func BenchmarkLayers(b *testing.B) {
	value, _ := Parse(phpBytes)

	b.Run("Parse", func(b *testing.B) {
		b.ReportAllocs()

		for b.Loop() {
			sinkValue, _ = Parse(phpBytes)
		}
	})

	b.Run("Decode", func(b *testing.B) {
		b.ReportAllocs()

		for b.Loop() {
			var u benchUser
			_ = Decode(value, &u)
			sinkUser = u
		}
	})

	b.Run("Encode", func(b *testing.B) {
		b.ReportAllocs()

		for b.Loop() {
			sinkValue, _ = Encode(benchStruct)
		}
	})

	b.Run("Write", func(b *testing.B) {
		encoded, _ := Encode(benchStruct)

		b.ReportAllocs()

		for b.Loop() {
			sinkBytes, _ = Write(encoded)
		}
	})
}
