package phpserialize

import (
	"reflect"
	"sort"
)

// field describes a struct field selected for encoding/decoding, after
// applying tags and embedded-struct promotion.
type field struct {
	name      string
	index     []int
	omitEmpty bool
}

// typeFields returns the list of fields that should be encoded for the given
// struct type, following the same rules as encoding/json: exported fields,
// `php` tag handling, promotion of anonymous (embedded) struct fields, and
// shallow-field-wins conflict resolution.
//
// It performs a breadth-first traversal so that shallower fields dominate
// deeper ones, and computes results fresh each call to avoid global mutable
// state.
func typeFields(t reflect.Type) []field {
	current := []embeddedType{}
	next := []embeddedType{{typ: t}}

	// visited records struct types seen at previous levels to avoid infinite
	// recursion on recursive types.
	visited := map[reflect.Type]bool{}

	var fields []field

	for len(next) > 0 {
		current, next = next, current[:0]
		count := map[reflect.Type]int{}

		for _, e := range current {
			if visited[e.typ] {
				continue
			}
			visited[e.typ] = true

			for i := 0; i < e.typ.NumField(); i++ {
				sf := e.typ.Field(i)
				if !fieldIsVisible(sf) {
					continue
				}
				tag := sf.Tag.Get("php")
				opts := parseTag(tag)
				if opts.skip {
					continue
				}

				index := make([]int, len(e.index)+1)
				copy(index, e.index)
				index[len(e.index)] = i

				ft := sf.Type
				if ft.Kind() == reflect.Pointer {
					ft = ft.Elem()
				}

				// Record a named field, or recurse into an anonymous struct.
				if opts.name != "" || !sf.Anonymous || ft.Kind() != reflect.Struct {
					name := opts.name
					tagged := isValidTagName(opts.name)
					if !tagged {
						name = sf.Name
					}
					fields = append(fields, field{
						name:      name,
						index:     index,
						omitEmpty: opts.omitEmpty,
					})
					if count[e.typ] > 1 {
						// Multiple embeddings of the same type at this level:
						// add a duplicate so it is later dropped by dominance.
						fields = append(fields, fields[len(fields)-1])
					}
					continue
				}

				// Anonymous struct without an explicit name: queue for the next
				// level so its fields are promoted.
				count[ft]++
				if count[ft] == 1 {
					next = append(next, embeddedType{typ: ft, index: index})
				}
			}
		}
	}

	sort.Slice(fields, func(i, j int) bool {
		x, y := fields[i], fields[j]
		if x.name != y.name {
			return x.name < y.name
		}
		return byIndexLess(x.index, y.index)
	})

	return dominantFields(fields)
}

// embeddedType is a struct type queued for traversal along with the index path
// that reaches it from the root struct.
type embeddedType struct {
	typ   reflect.Type
	index []int
}

// fieldIsVisible reports whether a struct field should be considered. Exported
// fields are always visible; unexported fields are visible only when they are
// anonymous embeddings of a struct type (so their exported fields can be
// promoted).
func fieldIsVisible(sf reflect.StructField) bool {
	if sf.Anonymous {
		t := sf.Type
		if t.Kind() == reflect.Pointer {
			t = t.Elem()
		}
		if !sf.IsExported() && t.Kind() != reflect.Struct {
			// Unexported non-struct embedded field: not usable.
			return false
		}
		return true
	}
	return sf.IsExported()
}

// byIndexLess orders two index paths lexicographically, shallower first.
func byIndexLess(a, b []int) bool {
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return len(a) < len(b)
}

// dominantFields removes fields that are shadowed by a shallower field of the
// same name. When exactly one field has the shallowest depth for a given name
// it dominates; ties are dropped entirely (matching encoding/json).
func dominantFields(fields []field) []field {
	out := fields[:0]
	for i := 0; i < len(fields); {
		// Gather the run of fields sharing this name (fields is name-sorted).
		name := fields[i].name
		j := i + 1
		for j < len(fields) && fields[j].name == name {
			j++
		}
		run := fields[i:j]
		i = j

		if len(run) == 1 {
			out = append(out, run[0])
			continue
		}
		if dominant, ok := dominant(run); ok {
			out = append(out, dominant)
		}
	}

	sort.Slice(out, func(i, j int) bool {
		return byIndexLess(out[i].index, out[j].index)
	})
	return out
}

// dominant selects the field with the shallowest index path in a run of
// same-named fields. It returns ok=false when there is a tie at the shallowest
// depth, in which case all such fields are ignored.
func dominant(run []field) (field, bool) {
	if len(run) == 1 {
		return run[0], true
	}
	// run is sorted by index (shallowest first) within a given name because the
	// caller sorted by name then index.
	sort.Slice(run, func(i, j int) bool {
		return byIndexLess(run[i].index, run[j].index)
	})
	if len(run[0].index) == len(run[1].index) {
		return field{}, false
	}
	return run[0], true
}

// fieldByIndex walks an index path from v, treating nil embedded pointers as
// absent (returning an invalid Value). Used when reading fields for encoding.
func fieldByIndex(v reflect.Value, index []int) reflect.Value {
	for i, x := range index {
		if i > 0 {
			if v.Kind() == reflect.Pointer {
				if v.IsNil() {
					return reflect.Value{}
				}
				v = v.Elem()
			}
		}
		v = v.Field(x)
	}
	return v
}

// fieldByIndexAlloc walks an index path from v, allocating embedded pointer
// structs as needed. Used when assigning fields during decoding.
func fieldByIndexAlloc(v reflect.Value, index []int) reflect.Value {
	for i, x := range index {
		if i > 0 {
			if v.Kind() == reflect.Pointer {
				if v.IsNil() {
					v.Set(reflect.New(v.Type().Elem()))
				}
				v = v.Elem()
			}
		}
		v = v.Field(x)
	}
	return v
}
