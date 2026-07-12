package phpserialize

import "strings"

// tagOptions holds the parsed contents of a `php:"..."` struct tag.
type tagOptions struct {
	// name is the field name to use in the serialized output. Empty means the
	// Go field name should be used.
	name string
	// omitEmpty reports whether the ",omitempty" option was present.
	omitEmpty bool
	// skip reports whether the tag was `php:"-"`, meaning the field is ignored.
	skip bool
}

// parseTag parses a struct field's `php` tag value following the same rules as
// encoding/json:
//
//   - "-" skips the field entirely (unless written as "-,").
//   - The first comma-separated element is the field name (may be empty).
//   - Subsequent elements are options; only "omitempty" is recognized.
func parseTag(tag string) tagOptions {
	if tag == "-" {
		return tagOptions{skip: true}
	}
	var opts tagOptions
	name, rest, hasComma := strings.Cut(tag, ",")
	opts.name = name
	if hasComma {
		for rest != "" {
			var opt string
			opt, rest, _ = strings.Cut(rest, ",")
			if opt == "omitempty" {
				opts.omitEmpty = true
			}
		}
	}
	return opts
}

// isValidTagName reports whether a tag-provided field name is acceptable. Like
// encoding/json, we accept a broad set of characters but disallow the empty
// string being treated as an override (empty falls back to the Go field name).
func isValidTagName(name string) bool {
	return name != ""
}
