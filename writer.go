package phpserialize

import (
	"math"
	"strconv"
)

// Write serializes an intermediate [Value] into canonical PHP serialized
// bytes. It is the exact inverse of [Parse]. String lengths are emitted as
// byte counts, using the UTF-8 bytes of Go strings verbatim.
func Write(v Value) ([]byte, error) {
	var w writer
	if err := w.writeValue(v); err != nil {
		return nil, err
	}
	return w.buf, nil
}

// writer accumulates serialized output.
type writer struct {
	buf []byte
}

func (w *writer) writeValue(v Value) error {
	switch v.Kind {
	case NullKind:
		w.buf = append(w.buf, 'N', ';')
	case BoolKind:
		w.buf = append(w.buf, 'b', ':')
		if v.Bool {
			w.buf = append(w.buf, '1')
		} else {
			w.buf = append(w.buf, '0')
		}
		w.buf = append(w.buf, ';')
	case IntKind:
		w.buf = append(w.buf, 'i', ':')
		w.buf = strconv.AppendInt(w.buf, v.Int, 10)
		w.buf = append(w.buf, ';')
	case FloatKind:
		w.buf = append(w.buf, 'd', ':')
		w.buf = appendPHPFloat(w.buf, v.Float)
		w.buf = append(w.buf, ';')
	case StringKind:
		w.writeString(v.String)
	case ArrayKind:
		return w.writeArray(v.Array)
	case ObjectKind:
		return w.writeObject(v.Object)
	default:
		return &UnsupportedValueError{Str: "unknown value kind " + v.Kind.String()}
	}
	return nil
}

// writeString emits s:<byte-len>:"<bytes>";.
func (w *writer) writeString(s string) {
	w.buf = append(w.buf, 's', ':')
	w.buf = strconv.AppendInt(w.buf, int64(len(s)), 10)
	w.buf = append(w.buf, ':', '"')
	w.buf = append(w.buf, s...)
	w.buf = append(w.buf, '"', ';')
}

// writeArray emits a:<count>:{ key value ... }.
func (w *writer) writeArray(entries []Entry) error {
	w.buf = append(w.buf, 'a', ':')
	w.buf = strconv.AppendInt(w.buf, int64(len(entries)), 10)
	w.buf = append(w.buf, ':', '{')
	for _, e := range entries {
		if !isValidArrayKey(e.Key.Kind) {
			return &UnsupportedValueError{Str: "invalid array key kind " + e.Key.Kind.String()}
		}
		if err := w.writeValue(e.Key); err != nil {
			return err
		}
		if err := w.writeValue(e.Value); err != nil {
			return err
		}
	}
	w.buf = append(w.buf, '}')
	return nil
}

// writeObject emits O:<name-len>:"class":<count>:{ name value ... }.
func (w *writer) writeObject(o *Object) error {
	if o == nil {
		return &UnsupportedValueError{Str: "nil object"}
	}
	w.buf = append(w.buf, 'O', ':')
	w.buf = strconv.AppendInt(w.buf, int64(len(o.ClassName)), 10)
	w.buf = append(w.buf, ':', '"')
	w.buf = append(w.buf, o.ClassName...)
	w.buf = append(w.buf, '"', ':')
	w.buf = strconv.AppendInt(w.buf, int64(len(o.Fields)), 10)
	w.buf = append(w.buf, ':', '{')
	for _, f := range o.Fields {
		w.writeString(f.Name)
		if err := w.writeValue(f.Value); err != nil {
			return err
		}
	}
	w.buf = append(w.buf, '}')
	return nil
}

// appendPHPFloat appends f to buf using the same formatting PHP's serialize()
// uses when serialize_precision is -1 (the default): the shortest decimal
// string that round-trips, laid out by PHP's php_gcvt algorithm.
func appendPHPFloat(buf []byte, f float64) []byte {
	switch {
	case math.IsInf(f, 1):
		return append(buf, "INF"...)
	case math.IsInf(f, -1):
		return append(buf, "-INF"...)
	case math.IsNaN(f):
		return append(buf, "NAN"...)
	}
	return append(buf, phpGcvt(f)...)
}

// phpGcvt reproduces PHP's php_gcvt(value, -1, '.', 'E', ...) used by
// serialize(). It obtains the shortest round-tripping digit string from Go's
// strconv and lays it out exactly as PHP does.
func phpGcvt(value float64) string {
	// Go's 'e' format with precision -1 yields the shortest representation as
	// "d.ddde±dd" (or "de±dd" with no fraction). From it we recover the digit
	// string and the decimal exponent that zend_dtoa would produce.
	rep := strconv.FormatFloat(value, 'e', -1, 64)

	neg := false
	if rep[0] == '-' {
		neg = true
		rep = rep[1:]
	}

	// Split mantissa and exponent.
	ePos := -1
	for i := 0; i < len(rep); i++ {
		if rep[i] == 'e' {
			ePos = i
			break
		}
	}
	mant := rep[:ePos]
	exp, _ := strconv.Atoi(rep[ePos+1:])

	// digits holds the significant digits with no decimal point.
	var digits string
	if dot := indexByte(mant, '.'); dot >= 0 {
		digits = mant[:dot] + mant[dot+1:]
	} else {
		digits = mant
	}

	// decpt is the position of the decimal point relative to the start of
	// digits (value = 0.digits * 10^decpt in zend_dtoa terms).
	decpt := exp + 1

	// ndigit is fixed at 17 for the shortest (mode 0) path in php_gcvt.
	const ndigit = 17

	var out []byte
	if neg {
		out = append(out, '-')
	}

	if decpt > ndigit || decpt < -3 {
		// Exponential format: "d.ddddE±XX".
		e := decpt - 1
		esign := byte('+')
		if e < 0 {
			esign = '-'
			e = -e
		}
		out = append(out, digits[0])
		out = append(out, '.')
		if len(digits) == 1 {
			out = append(out, '0')
		} else {
			out = append(out, digits[1:]...)
		}
		out = append(out, 'E', esign)
		out = strconv.AppendInt(out, int64(e), 10)
	} else if decpt <= 0 {
		// Fractional format: "0.00ddd".
		out = append(out, '0', '.')
		for i := 0; i < -decpt; i++ {
			out = append(out, '0')
		}
		out = append(out, digits...)
	} else {
		// Standard format, possibly with a fractional part.
		for i := 0; i < decpt; i++ {
			if i < len(digits) {
				out = append(out, digits[i])
			} else {
				out = append(out, '0')
			}
		}
		if decpt < len(digits) {
			out = append(out, '.')
			out = append(out, digits[decpt:]...)
		}
	}
	return string(out)
}

// indexByte reports the index of the first occurrence of c in s, or -1.
func indexByte(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}
