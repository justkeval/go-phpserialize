package phpserialize

import (
	"math"
	"strconv"
)

// Parse parses PHP serialized bytes into a [Value]. It performs a strict
// recursive-descent parse and never inspects any Go destination type.
//
// The entire input must be consumed by a single serialized value; trailing
// bytes are reported as a [SyntaxError].
func Parse(data []byte) (Value, error) {
	p := &parser{data: data}
	v, err := p.parseValue()
	if err != nil {
		return Value{}, err
	}
	if p.pos != len(p.data) {
		return Value{}, p.errorf("unexpected trailing data")
	}
	return v, nil
}

// parser holds the input and the current read position.
type parser struct {
	data []byte
	pos  int
}

// errorf builds a SyntaxError anchored at the current position.
func (p *parser) errorf(msg string) *SyntaxError {
	return &SyntaxError{msg: msg, Offset: int64(p.pos)}
}

// errorAt builds a SyntaxError anchored at a specific offset.
func (p *parser) errorAt(msg string, off int) *SyntaxError {
	return &SyntaxError{msg: msg, Offset: int64(off)}
}

// peek returns the byte at the current position, or reports whether input
// remains.
func (p *parser) peek() (byte, bool) {
	if p.pos >= len(p.data) {
		return 0, false
	}
	return p.data[p.pos], true
}

// expect consumes a single byte, requiring it to equal c.
func (p *parser) expect(c byte) error {
	b, ok := p.peek()
	if !ok {
		return p.errorf("unexpected end of input, expected " + strconv.QuoteRune(rune(c)))
	}
	if b != c {
		return p.errorf("expected " + strconv.QuoteRune(rune(c)) + ", found " + strconv.QuoteRune(rune(b)))
	}
	p.pos++
	return nil
}

// parseValue parses a single serialized value at the current position.
func (p *parser) parseValue() (Value, error) {
	b, ok := p.peek()
	if !ok {
		return Value{}, p.errorf("unexpected end of input")
	}
	switch b {
	case 'N':
		return p.parseNull()
	case 'b':
		return p.parseBool()
	case 'i':
		return p.parseInt()
	case 'd':
		return p.parseFloat()
	case 's':
		return p.parseString()
	case 'a':
		return p.parseArray()
	case 'O':
		return p.parseObject()
	case 'C', 'R', 'r':
		// Custom serialized objects and references are intentionally
		// unsupported.
		return Value{}, ErrUnsupportedType
	default:
		return Value{}, p.errorf("unexpected type marker " + strconv.QuoteRune(rune(b)))
	}
}

// parseNull parses "N;".
func (p *parser) parseNull() (Value, error) {
	p.pos++ // consume 'N'
	if err := p.expect(';'); err != nil {
		return Value{}, err
	}
	return Value{Kind: NullKind}, nil
}

// parseBool parses "b:0;" or "b:1;".
func (p *parser) parseBool() (Value, error) {
	p.pos++ // consume 'b'
	if err := p.expect(':'); err != nil {
		return Value{}, err
	}
	b, ok := p.peek()
	if !ok {
		return Value{}, p.errorf("unexpected end of input in bool")
	}
	var val bool
	switch b {
	case '0':
		val = false
	case '1':
		val = true
	default:
		return Value{}, p.errorf("invalid bool value " + strconv.QuoteRune(rune(b)))
	}
	p.pos++
	if err := p.expect(';'); err != nil {
		return Value{}, err
	}
	return Value{Kind: BoolKind, Bool: val}, nil
}

// parseInt parses "i:<digits>;".
func (p *parser) parseInt() (Value, error) {
	p.pos++ // consume 'i'
	if err := p.expect(':'); err != nil {
		return Value{}, err
	}
	raw, err := p.readUntil(';')
	if err != nil {
		return Value{}, err
	}
	n, perr := strconv.ParseInt(string(raw), 10, 64)
	if perr != nil {
		return Value{}, p.errorAt("malformed integer "+strconv.Quote(string(raw)), p.pos-len(raw)-1)
	}
	return Value{Kind: IntKind, Int: n}, nil
}

// parseFloat parses "d:<number>;". It also accepts the PHP spellings of the
// special values INF, -INF and NAN.
func (p *parser) parseFloat() (Value, error) {
	p.pos++ // consume 'd'
	if err := p.expect(':'); err != nil {
		return Value{}, err
	}
	start := p.pos
	raw, err := p.readUntil(';')
	if err != nil {
		return Value{}, err
	}
	s := string(raw)
	switch s {
	case "INF":
		return Value{Kind: FloatKind, Float: math.Inf(1)}, nil
	case "-INF":
		return Value{Kind: FloatKind, Float: math.Inf(-1)}, nil
	case "NAN":
		return Value{Kind: FloatKind, Float: math.NaN()}, nil
	}
	f, perr := strconv.ParseFloat(s, 64)
	if perr != nil {
		return Value{}, p.errorAt("malformed double "+strconv.Quote(s), start)
	}
	return Value{Kind: FloatKind, Float: f}, nil
}

// parseString parses "s:<len>:\"<bytes>\";".
func (p *parser) parseString() (Value, error) {
	p.pos++ // consume 's'
	s, err := p.parseRawString()
	if err != nil {
		return Value{}, err
	}
	return Value{Kind: StringKind, String: s}, nil
}

// parseRawString parses the ":<len>:\"<bytes>\";" portion following an 's'
// marker and returns the raw string contents.
func (p *parser) parseRawString() (string, error) {
	if err := p.expect(':'); err != nil {
		return "", err
	}
	length, err := p.readLength()
	if err != nil {
		return "", err
	}
	if err := p.expect(':'); err != nil {
		return "", err
	}
	if err := p.expect('"'); err != nil {
		return "", err
	}
	if p.pos+length > len(p.data) {
		return "", p.errorf("string length exceeds input")
	}
	s := string(p.data[p.pos : p.pos+length])
	p.pos += length
	if err := p.expect('"'); err != nil {
		return "", err
	}
	if err := p.expect(';'); err != nil {
		return "", err
	}
	return s, nil
}

// parseArray parses "a:<count>:{ (key value)* }".
func (p *parser) parseArray() (Value, error) {
	p.pos++ // consume 'a'
	if err := p.expect(':'); err != nil {
		return Value{}, err
	}
	count, err := p.readLength()
	if err != nil {
		return Value{}, err
	}
	if err := p.expect(':'); err != nil {
		return Value{}, err
	}
	if err := p.expect('{'); err != nil {
		return Value{}, err
	}
	entries := make([]Entry, 0, count)
	for i := 0; i < count; i++ {
		key, err := p.parseValue()
		if err != nil {
			return Value{}, err
		}
		if !isValidArrayKey(key.Kind) {
			return Value{}, p.errorf("invalid array key type " + key.Kind.String())
		}
		val, err := p.parseValue()
		if err != nil {
			return Value{}, err
		}
		entries = append(entries, Entry{Key: key, Value: val})
	}
	if err := p.expect('}'); err != nil {
		return Value{}, err
	}
	return Value{Kind: ArrayKind, Array: entries}, nil
}

// parseObject parses "O:<len>:\"class\":<count>:{ (name value)* }".
func (p *parser) parseObject() (Value, error) {
	p.pos++ // consume 'O'
	if err := p.expect(':'); err != nil {
		return Value{}, err
	}
	nameLen, err := p.readLength()
	if err != nil {
		return Value{}, err
	}
	if err := p.expect(':'); err != nil {
		return Value{}, err
	}
	if err := p.expect('"'); err != nil {
		return Value{}, err
	}
	if p.pos+nameLen > len(p.data) {
		return Value{}, p.errorf("class name length exceeds input")
	}
	className := string(p.data[p.pos : p.pos+nameLen])
	p.pos += nameLen
	if err := p.expect('"'); err != nil {
		return Value{}, err
	}
	if err := p.expect(':'); err != nil {
		return Value{}, err
	}
	count, err := p.readLength()
	if err != nil {
		return Value{}, err
	}
	if err := p.expect(':'); err != nil {
		return Value{}, err
	}
	if err := p.expect('{'); err != nil {
		return Value{}, err
	}
	fields := make([]Field, 0, count)
	for i := 0; i < count; i++ {
		nameVal, err := p.parseValue()
		if err != nil {
			return Value{}, err
		}
		if nameVal.Kind != StringKind {
			return Value{}, p.errorf("object property name must be a string, found " + nameVal.Kind.String())
		}
		val, err := p.parseValue()
		if err != nil {
			return Value{}, err
		}
		fields = append(fields, Field{Name: nameVal.String, Value: val})
	}
	if err := p.expect('}'); err != nil {
		return Value{}, err
	}
	return Value{Kind: ObjectKind, Object: &Object{ClassName: className, Fields: fields}}, nil
}

// readUntil reads bytes up to (but not consuming) the delimiter, then consumes
// the delimiter. It returns the bytes read, which may be empty.
func (p *parser) readUntil(delim byte) ([]byte, error) {
	start := p.pos
	for p.pos < len(p.data) {
		if p.data[p.pos] == delim {
			raw := p.data[start:p.pos]
			p.pos++ // consume delimiter
			return raw, nil
		}
		p.pos++
	}
	return nil, p.errorAt("unexpected end of input, expected "+strconv.QuoteRune(rune(delim)), start)
}

// readLength reads a non-negative decimal length terminated by ':' (which it
// does not consume). It rejects signs, empty numbers and overflow.
func (p *parser) readLength() (int, error) {
	start := p.pos
	for p.pos < len(p.data) {
		c := p.data[p.pos]
		if c < '0' || c > '9' {
			break
		}
		p.pos++
	}
	if p.pos == start {
		return 0, p.errorf("expected length digits")
	}
	n, err := strconv.Atoi(string(p.data[start:p.pos]))
	if err != nil || n < 0 {
		return 0, p.errorAt("invalid length", start)
	}
	return n, nil
}

// isValidArrayKey reports whether a parsed key kind is legal for a PHP array.
// PHP arrays are keyed only by integers and strings.
func isValidArrayKey(k Kind) bool {
	return k == IntKind || k == StringKind
}
