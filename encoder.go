package sonnet

import (
	"bytes"
	"encoding"
	"errors"
	"github.com/sugawarayuuta/sonnet/internal/mem"
	"io"
	"reflect"
	"strconv"
)

type (
	Encoder struct {
		out    io.Writer
		html   bool
		level  uint
		prefix string
		indent string
		seen   map[any]struct{}
	}
	encoder func([]byte, reflect.Value, *Encoder) ([]byte, error)
)

var (
	encs = makeCache[reflect.Type, encoder]()
	lens = makeCache[reflect.Type, int]()
)

const (
	maxCycles = 1000
)

// Marshal returns the JSON encoding of v.
//
// Marshal traverses the value v recursively.
// If an encountered value implements the Marshaler interface
// and is not a nil pointer, Marshal calls its MarshalJSON method
// to produce JSON. If no MarshalJSON method is present but the
// value implements encoding.TextMarshaler instead, Marshal calls
// its MarshalText method and encodes the result as a JSON string.
// The nil pointer exception is not strictly necessary
// but mimics a similar, necessary exception in the behavior of
// UnmarshalJSON.
//
// Otherwise, Marshal uses the following type-dependent default encodings:
//
// Boolean values encode as JSON booleans.
//
// Floating point, integer, and Number values encode as JSON numbers.
// NaN and +/-Inf values will return an [UnsupportedValueError].
//
// String values encode as JSON strings coerced to valid UTF-8,
// replacing invalid bytes with the Unicode replacement rune.
// So that the JSON will be safe to embed inside HTML <script> tags,
// the string is encoded using HTMLEscape,
// which replaces "<", ">", "&", U+2028, and U+2029 are escaped
// to "\u003c","\u003e", "\u0026", "\u2028", and "\u2029".
// This replacement can be disabled when using an Encoder,
// by calling SetEscapeHTML(false).
//
// Array and slice values encode as JSON arrays, except that
// []byte encodes as a base64-encoded string, and a nil slice
// encodes as the null JSON value.
//
// Struct values encode as JSON objects.
// Each exported struct field becomes a member of the object, using the
// field name as the object key, unless the field is omitted for one of the
// reasons given below.
//
// The encoding of each struct field can be customized by the format string
// stored under the "json" key in the struct field's tag.
// The format string gives the name of the field, possibly followed by a
// comma-separated list of options. The name may be empty in order to
// specify options without overriding the default field name.
//
// The "omitempty" option specifies that the field should be omitted
// from the encoding if the field has an empty value, defined as
// false, 0, a nil pointer, a nil interface value, and any empty array,
// slice, map, or string.
//
// As a special case, if the field tag is "-", the field is always omitted.
// Note that a field with name "-" can still be generated using the tag "-,".
//
// Examples of struct field tags and their meanings:
//
//	// Field appears in JSON as key "myName".
//	Field int `json:"myName"`
//
//	// Field appears in JSON as key "myName" and
//	// the field is omitted from the object if its value is empty,
//	// as defined above.
//	Field int `json:"myName,omitempty"`
//
//	// Field appears in JSON as key "Field" (the default), but
//	// the field is skipped if empty.
//	// Note the leading comma.
//	Field int `json:",omitempty"`
//
//	// Field is ignored by this package.
//	Field int `json:"-"`
//
//	// Field appears in JSON as key "-".
//	Field int `json:"-,"`
//
// The "string" option signals that a field is stored as JSON inside a
// JSON-encoded string. It applies only to fields of string, floating point,
// integer, or boolean types. This extra level of encoding is sometimes used
// when communicating with JavaScript programs:
//
//	Int64String int64 `json:",string"`
//
// The key name will be used if it's a non-empty string consisting of
// only Unicode letters, digits, and ASCII punctuation except quotation
// marks, backslash, and comma.
//
// Anonymous struct fields are usually marshaled as if their inner exported fields
// were fields in the outer struct, subject to the usual Go visibility rules amended
// as described in the next paragraph.
// An anonymous struct field with a name given in its JSON tag is treated as
// having that name, rather than being anonymous.
// An anonymous struct field of interface type is treated the same as having
// that type as its name, rather than being anonymous.
//
// The Go visibility rules for struct fields are amended for JSON when
// deciding which field to marshal or unmarshal. If there are
// multiple fields at the same level, and that level is the least
// nested (and would therefore be the nesting level selected by the
// usual Go rules), the following extra rules apply:
//
// 1) Of those fields, if any are JSON-tagged, only tagged fields are considered,
// even if there are multiple untagged fields that would otherwise conflict.
//
// 2) If there is exactly one field (tagged or not according to the first rule), that is selected.
//
// 3) Otherwise there are multiple fields, and all are ignored; no error occurs.
//
// Handling of anonymous struct fields is new in Go 1.1.
// Prior to Go 1.1, anonymous struct fields were ignored. To force ignoring of
// an anonymous struct field in both current and earlier versions, give the field
// a JSON tag of "-".
//
// Map values encode as JSON objects. The map's key type must either be a
// string, an integer type, or implement encoding.TextMarshaler. The map keys
// are sorted and used as JSON object keys by applying the following rules,
// subject to the UTF-8 coercion described for string values above:
//   - keys of any string type are used directly
//   - encoding.TextMarshalers are marshaled
//   - integer keys are converted to strings
//
// Pointer values encode as the value pointed to.
// A nil pointer encodes as the null JSON value.
//
// Interface values encode as the value contained in the interface.
// A nil interface value encodes as the null JSON value.
//
// Channel, complex, and function values cannot be encoded in JSON.
// Attempting to encode such a value causes Marshal to return
// an UnsupportedTypeError.
//
// JSON cannot represent cyclic data structures and Marshal does not
// handle them. Passing cyclic structures to Marshal will result in
// an error.
func Marshal(val any) ([]byte, error) {
	enc := Encoder{html: true}
	return enc.encode(val)
}

// MarshalIndent is like Marshal but applies Indent to format the output.
// Each JSON element in the output will begin on a new line beginning with prefix
// followed by one or more copies of indent according to the indentation nesting.
func MarshalIndent(val any, prefix, indent string) ([]byte, error) {
	dst, err := Marshal(val)
	if err == nil {
		var buf bytes.Buffer
		buf.Grow(len(dst) * 2)
		err = Indent(&buf, dst, prefix, indent)
		dst = buf.Bytes()
	}
	return dst, err
}

// NewEncoder returns a new encoder that writes to w.
func NewEncoder(out io.Writer) *Encoder {
	return &Encoder{out: out}
}

// Encode writes the JSON encoding of v to the stream,
// followed by a newline character.
//
// See the documentation for Marshal for details about the
// conversion of Go values to JSON.
func (enc *Encoder) Encode(val any) error {
	dst, err := enc.encode(val)
	if err != nil {
		return err
	}
	if enc.prefix != "" || enc.indent != "" {
		var buf bytes.Buffer
		buf.Grow(len(dst) * 2)
		err = Indent(&buf, dst, enc.prefix, enc.indent)
		if err != nil {
			return err
		}
		dst = buf.Bytes()
	}
	wrt, err := enc.out.Write(dst)
	if err != nil {
		return err
	}
	if wrt != len(dst) {
		return io.ErrShortWrite
	}
	mem.Put(dst)
	return nil
}

// SetEscapeHTML specifies whether problematic HTML characters
// should be escaped inside JSON quoted strings.
// The default behavior is to escape &, <, and > to \u0026, \u003c, and \u003e
// to avoid certain safety problems that can arise when embedding JSON in HTML.
//
// In non-HTML settings where the escaping interferes with the readability
// of the output, SetEscapeHTML(false) disables this behavior.
func (enc *Encoder) SetEscapeHTML(html bool) {
	enc.html = html
}

// SetIndent instructs the encoder to format each subsequent encoded
// value as if indented by the package-level function Indent(dst, src, prefix, indent).
// Calling SetIndent("", "") disables indentation.
func (enc *Encoder) SetIndent(prefix, indent string) {
	enc.prefix = prefix
	enc.indent = indent
}

// Compact appends to dst the JSON-encoded src with
// insignificant space characters elided.
func Compact(dst *bytes.Buffer, src []byte) error {
	dst.Grow(len(src))
	buf := dst.AvailableBuffer()
	comp := compactor{
		dst:  buf,
		src:  src,
		html: false,
	}
	comp.eatSpaces()
	if len(src) <= comp.read {
		return comp.errSyntax("unexpected EOF reading a byte")
	}
	head := src[comp.read]
	comp.read++
	err := comp.compact(head)
	if err != nil {
		return err
	}
	if comp.write < comp.read {
		comp.dst = append(comp.dst, comp.src[comp.write:comp.read]...)
	}
	dst.Write(comp.dst)
	return nil
}

// HTMLEscape appends to dst the JSON-encoded src with <, >, &, U+2028 and U+2029
// characters inside string literals changed to \u003c, \u003e, \u0026, \u2028, \u2029
// so that the JSON will be safe to embed inside HTML <script> tags.
// For historical reasons, web browsers don't honor standard HTML
// escaping within <script> tags, so an alternative JSON encoding must
// be used.
func HTMLEscape(dst *bytes.Buffer, src []byte) {
	const hex = "0123456789abcdef"
	dst.Grow(len(src))
	buf := dst.AvailableBuffer()
	// The characters can only appear in string literals,
	// so just scan the string one byte at a time.
	start := 0
	for idx, char := range src {
		if char == '<' || char == '>' || char == '&' {
			buf = append(buf, src[start:idx]...)
			buf = append(buf, '\\', 'u', '0', '0', hex[char>>4], hex[char&0xF])
			start = idx + 1
		}
		// Convert U+2028 and U+2029 (E2 80 A8 and E2 80 A9).
		if char == 0xE2 && idx+2 < len(src) && src[idx+1] == 0x80 && src[idx+2]&^1 == 0xA8 {
			buf = append(buf, src[start:idx]...)
			buf = append(buf, '\\', 'u', '2', '0', '2', hex[src[idx+2]&0xF])
			start = idx + len("\u2029")
		}
	}
	dst.Write(append(buf, src[start:]...))
}

// Indent appends to dst an indented form of the JSON-encoded src.
// Each element in a JSON object or array begins on a new,
// indented line beginning with prefix followed by one or more
// copies of indent according to the indentation nesting.
// The data appended to dst does not begin with the prefix nor
// any indentation, to make it easier to embed inside other formatted JSON data.
// Although leading space characters (space, tab, carriage return, newline)
// at the beginning of src are dropped, trailing space characters
// at the end of src are preserved and copied to dst.
// For example, if src has no trailing spaces, neither will dst;
// if src ends in a trailing newline, so will dst.
func Indent(dst *bytes.Buffer, src []byte, prefix, indent string) error {
	dst.Grow(len(src) * 2)
	buf := dst.AvailableBuffer()
	comp := compactor{
		dst:    buf,
		src:    src,
		html:   false,
		prefix: prefix,
		indent: indent,
	}
	comp.eatSpaces()
	if len(src) <= comp.read {
		return comp.errSyntax("unexpected EOF reading a byte")
	}
	head := src[comp.read]
	comp.read++
	err := comp.compact(head)
	if err != nil {
		return err
	}
	if comp.write < comp.read {
		comp.dst = append(comp.dst, comp.src[comp.write:comp.read]...)
	}
	dst.Write(comp.dst)
	return nil
}

func (enc *Encoder) encode(val any) ([]byte, error) {
	ref := reflect.ValueOf(val)
	typ := reflect.TypeOf(val)
	if val == nil {
		return []byte("null"), nil
	}
	num, ok := lens.get(typ)
	if !ok {
		num = 1 << 10
	}
	dst := mem.Get(num)[:0]
	fnc, ok := encs.get(typ)
	if !ok {
		fnc = compileEncoder(typ, true)
		encs.set(typ, fnc)
	}
	dst, err := fnc(dst, ref, enc)
	if err != nil {
		return nil, err
	}
	num += len(dst)
	lens.set(typ, (num+num&1)>>1)
	return dst, nil
}

func compileEncoder(typ reflect.Type, addr bool) encoder {
	const lenInt = 5   // int, int8, int16, int32, int64
	const lenUint = 6  // uint, uint8, uint16, uint32, uint64, uintptr
	const lenFloat = 2 // float32, float64
	kind := typ.Kind()
	ptr := reflect.PointerTo(typ)
	if addr && kind != reflect.Pointer && ptr.Implements(marshaler) {
		return compilePointerMarshalerEncoder(typ)
	}
	if addr && kind != reflect.Pointer && ptr.Implements(textMarshaler) {
		return compilePointerTextMarshalerEncoder(typ)
	}
	if typ.Implements(marshaler) {
		return encodeMarshaler
	}
	if typ.Implements(textMarshaler) {
		return encodeTextMarshaler
	}
	if typ == number {
		return encodeNumber
	}
	if kind == reflect.String {
		return encodeString
	}
	if kind-reflect.Int < lenInt {
		return encodeInt
	}
	if kind-reflect.Uint < lenUint {
		return encodeUint
	}
	if kind-reflect.Float32 < lenFloat {
		return encodeFloat
	}
	if kind == reflect.Bool {
		return encodeBool
	}
	if kind == reflect.Interface {
		return encodeInterface
	}
	if kind == reflect.Array {
		return compileArrayEncoder(typ)
	}
	if kind == reflect.Slice {
		return compileSliceEncoder(typ)
	}
	if kind == reflect.Map {
		return compileMapEncoder(typ)
	}
	if kind == reflect.Struct {
		return compileStructEncoder(typ)
	}
	if kind == reflect.Pointer {
		return compilePointerEncoder(typ)
	}
	return encodeUnsupported
}

func encodeMarshaler(dst []byte, val reflect.Value, enc *Encoder) ([]byte, error) {
	const fnc = "MarshalJSON"
	if val.Kind() == reflect.Pointer && val.IsNil() {
		return append(dst, "null"...), nil
	}
	mar, ok := val.Interface().(Marshaler)
	if !ok {
		return append(dst, "null"...), nil
	}
	src, err := mar.MarshalJSON()
	if err != nil {
		return nil, &MarshalerError{
			Type:       val.Type(),
			Err:        err,
			sourceFunc: fnc,
		}
	}

	comp := compactor{
		dst:  dst,
		src:  src,
		html: enc.html,
	}
	comp.eatSpaces()
	if len(src) <= comp.read {
		return nil, &MarshalerError{
			Type:       val.Type(),
			Err:        comp.errSyntax("unexpected EOF reading a byte"),
			sourceFunc: fnc,
		}
	}
	head := src[comp.read]
	comp.read++

	err = comp.compact(head)
	if err != nil {
		return nil, err
	}
	if comp.write < comp.read {
		comp.dst = append(comp.dst, comp.src[comp.write:comp.read]...)
	}
	return comp.dst, nil
}

func encodeTextMarshaler(dst []byte, val reflect.Value, enc *Encoder) ([]byte, error) {
	const fnc = "MarshalText"
	if val.Kind() == reflect.Pointer && val.IsNil() {
		return append(dst, "null"...), nil
	}
	mar, ok := val.Interface().(encoding.TextMarshaler)
	if !ok {
		return append(dst, "null"...), nil
	}
	src, err := mar.MarshalText()
	if err != nil {
		return nil, &MarshalerError{
			Type:       val.Type(),
			Err:        err,
			sourceFunc: fnc,
		}
	}
	return appendString(dst, string(src), enc.html), nil
}

func encodeNumber(dst []byte, val reflect.Value, enc *Encoder) ([]byte, error) {
	num := val.String()
	if num == "" {
		num = "0"
	}
	if !isValidNumber(num) {
		return nil, errors.New("sonnet: invalid number literal: " + strconv.Quote(num))
	}
	return append(dst, num...), nil
}

func encodeString(dst []byte, val reflect.Value, enc *Encoder) ([]byte, error) {
	return appendString(dst, val.String(), enc.html), nil
}

func encodeInt(dst []byte, val reflect.Value, enc *Encoder) ([]byte, error) {
	i64 := val.Int()
	if i64 < 0 {
		return append(dst, fmtInt(uint64(-i64), true)...), nil
	}
	return append(dst, fmtInt(uint64(i64), false)...), nil
}

func encodeUint(dst []byte, val reflect.Value, enc *Encoder) ([]byte, error) {
	u64 := val.Uint()
	return append(dst, fmtInt(u64, false)...), nil
}

func encodeFloat(dst []byte, val reflect.Value, enc *Encoder) ([]byte, error) {
	return appendFloat(dst, val.Float(), val.Type().Bits())
}

func encodeBool(dst []byte, val reflect.Value, enc *Encoder) ([]byte, error) {
	if val.Bool() {
		return append(dst, "true"...), nil
	}
	return append(dst, "false"...), nil
}

func encodeInterface(dst []byte, val reflect.Value, enc *Encoder) ([]byte, error) {
	return appendAny(dst, val.Interface(), enc)
}

func encodeUnsupported(dst []byte, val reflect.Value, enc *Encoder) ([]byte, error) {
	return nil, &UnsupportedTypeError{Type: val.Type()}
}
