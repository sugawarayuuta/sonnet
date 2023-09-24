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
	fieldError string
	decoder    func(byte, reflect.Value, *Decoder) error
)

var (
	decs = makeCache[reflect.Type, decoder]()
)

func (err fieldError) Error() string {
	return "sonnet: " + string(err)
}

// NewDecoder returns a new decoder that reads from r.
//
// The decoder introduces its own buffering and may
// read data from r beyond the JSON values requested.
func NewDecoder(inp io.Reader) *Decoder {
	return &Decoder{
		buf: mem.Get(1 << 10)[:0],
		inp: inp,
	}
}

// Buffered returns a reader of the data remaining in the Decoder's
// buffer. The reader is valid until the next call to Decode.
func (dec *Decoder) Buffered() io.Reader {
	return bytes.NewReader(dec.buf[dec.pos:])
}

// More reports whether there is another element in the
// current array or object being parsed.
func (dec *Decoder) More() bool {
	dec.eatSpaces()
	if dec.pos >= len(dec.buf) && !dec.fill() {
		return false
	}
	head := dec.buf[dec.pos]
	return head != ']' && head != '}'
}

// Valid reports whether data is a valid JSON encoding.
func Valid(inp []byte) bool {
	dec := Decoder{
		buf: inp,
	}
	dec.eatSpaces()
	if len(dec.buf) <= dec.pos {
		return false
	}
	head := dec.buf[dec.pos]
	dec.pos++
	err := dec.skip(head)
	if err != nil {
		return false
	}
	dec.eatSpaces()
	return dec.pos >= len(dec.buf)
}

func (dec *Decoder) decode(val any) error {
	ref := reflect.ValueOf(val)
	typ := reflect.TypeOf(val)
	if ref.Kind() != reflect.Pointer || ref.IsNil() {
		// Kind check should be first, otherwise IsNil panics
		// because of a non-pointer value.
		return &InvalidUnmarshalError{Type: typ}
	}
	elm := typ.Elem() // known to be a pointer.
	fnc, ok := decs.get(elm)
	if !ok {
		fnc = compileDecoder(elm)
		decs.set(elm, fnc)
	}
	dec.eatSpaces()
	if dec.pos >= len(dec.buf) && !dec.fill() {
		return dec.errSyntax("unexpected EOF reading a byte")
	}
	head := dec.buf[dec.pos]
	dec.pos++
	err := addPointer(fnc(head, ref.Elem(), dec))
	if err != nil {
		return err
	}
	dec.eatSpaces()
	return nil
}

func addPointer(err error) error {
	// the encoding/json library checks type mismatches
	// of TextUnmarshaler's before applying
	// dereferenced values. mimic the handling.
	if err, ok := err.(*UnmarshalTypeError); ok {
		ptr := reflect.PointerTo(err.Type)
		if ptr.Implements(textUnmarshaler) {
			err.Type = ptr
		}
	}
	return err
}

// InputOffset returns the input stream byte offset of the current decoder position.
// The offset gives the location of the end of the most recently returned token
// and the beginning of the next token.
func (dec *Decoder) InputOffset() int64 {
	return int64(dec.prev + dec.pos)
}

// DisallowUnknownFields causes the Decoder to return an error when the destination
// is a struct and the input contains object keys which do not match any
// non-ignored, exported fields in the destination.
func (dec *Decoder) DisallowUnknownFields() {
	dec.opt |= optUnknownFields
}

// UseNumber causes the Decoder to unmarshal a number into an interface{} as a
// Number instead of as a float64.
func (dec *Decoder) UseNumber() {
	dec.opt |= optNumber
}

// Decode reads the next JSON-encoded value from its
// input and stores it in the value pointed to by v.
//
// See the documentation for Unmarshal for details about
// the conversion of JSON into a Go value.
func (dec *Decoder) Decode(val any) error {
	return dec.decode(val)
}

// Unmarshal parses the JSON-encoded data and stores the result
// in the value pointed to by v. If v is nil or not a pointer,
// Unmarshal returns an InvalidUnmarshalError.
//
// Unmarshal uses the inverse of the encodings that
// Marshal uses, allocating maps, slices, and pointers as necessary,
// with the following additional rules:
//
// To unmarshal JSON into a pointer, Unmarshal first handles the case of
// the JSON being the JSON literal null. In that case, Unmarshal sets
// the pointer to nil. Otherwise, Unmarshal unmarshals the JSON into
// the value pointed at by the pointer. If the pointer is nil, Unmarshal
// allocates a new value for it to point to.
//
// To unmarshal JSON into a value implementing the Unmarshaler interface,
// Unmarshal calls that value's UnmarshalJSON method, including
// when the input is a JSON null.
// Otherwise, if the value implements encoding.TextUnmarshaler
// and the input is a JSON quoted string, Unmarshal calls that value's
// UnmarshalText method with the unquoted form of the string.
//
// To unmarshal JSON into a struct, Unmarshal matches incoming object
// keys to the keys used by Marshal (either the struct field name or its tag),
// preferring an exact match but also accepting a case-insensitive match. By
// default, object keys which don't have a corresponding struct field are
// ignored (see Decoder.DisallowUnknownFields for an alternative).
//
// To unmarshal JSON into an interface value,
// Unmarshal stores one of these in the interface value:
//
//	bool, for JSON booleans
//	float64, for JSON numbers
//	string, for JSON strings
//	[]interface{}, for JSON arrays
//	map[string]interface{}, for JSON objects
//	nil for JSON null
//
// To unmarshal a JSON array into a slice, Unmarshal resets the slice length
// to zero and then appends each element to the slice.
// As a special case, to unmarshal an empty JSON array into a slice,
// Unmarshal replaces the slice with a new empty slice.
//
// To unmarshal a JSON array into a Go array, Unmarshal decodes
// JSON array elements into corresponding Go array elements.
// If the Go array is smaller than the JSON array,
// the additional JSON array elements are discarded.
// If the JSON array is smaller than the Go array,
// the additional Go array elements are set to zero values.
//
// To unmarshal a JSON object into a map, Unmarshal first establishes a map to
// use. If the map is nil, Unmarshal allocates a new map. Otherwise Unmarshal
// reuses the existing map, keeping existing entries. Unmarshal then stores
// key-value pairs from the JSON object into the map. The map's key type must
// either be any string type, an integer, implement json.Unmarshaler, or
// implement encoding.TextUnmarshaler.
//
// If the JSON-encoded data contain a syntax error, Unmarshal returns a SyntaxError.
//
// If a JSON value is not appropriate for a given target type,
// or if a JSON number overflows the target type, Unmarshal
// skips that field and completes the unmarshaling as best it can.
// If no more serious errors are encountered, Unmarshal returns
// an UnmarshalTypeError describing the earliest such error. In any
// case, it's not guaranteed that all the remaining fields following
// the problematic one will be unmarshaled into the target object.
//
// The JSON null value unmarshals into an interface, map, pointer, or slice
// by setting that Go value to nil. Because null is often used in JSON to mean
// “not present,” unmarshaling a JSON null into any other Go type has no effect
// on the value and produces no error.
//
// When unmarshaling quoted strings, invalid UTF-8 or
// invalid UTF-16 surrogate pairs are not treated as an error.
// Instead, they are replaced by the Unicode replacement
// character U+FFFD.
func Unmarshal(inp []byte, val any) error {
	dec := Decoder{
		buf: inp,
	}
	err := dec.decode(val)
	if err == nil && dec.pos < len(dec.buf) {
		err = dec.errSyntax("invalid character " + strconv.QuoteRune(rune(dec.buf[dec.pos])) + " after top-level value")
	}
	return err
}

func compileDecoder(typ reflect.Type) decoder {
	const lenInt = 5   // int, int8, int16, int32, int64
	const lenUint = 6  // uint, uint8, uint16, uint32, uint64, uintptr
	const lenFloat = 2 // float32, float64
	kind := typ.Kind()
	ptr := reflect.PointerTo(typ)
	if kind != reflect.Pointer && ptr.Implements(unmarshaler) {
		return decodeUnmarshaler
	}
	if kind != reflect.Pointer && ptr.Implements(textUnmarshaler) {
		return decodeTextUnmarshaler
	}
	if typ == number {
		return decodeNumber
	}
	if kind == reflect.String {
		return decodeString
	}
	if kind-reflect.Int < lenInt {
		return decodeInt
	}
	if kind-reflect.Uint < lenUint {
		return decodeUint
	}
	if kind-reflect.Float32 < lenFloat {
		return decodeFloat
	}
	if kind == reflect.Bool {
		return decodeBool
	}
	if kind == reflect.Interface {
		return decodeInterface
	}
	if kind == reflect.Array {
		return compileArrayDecoder(typ)
	}
	if kind == reflect.Slice {
		return compileSliceDecoder(typ)
	}
	if kind == reflect.Map {
		return compileMapDecoder(typ)
	}
	if kind == reflect.Struct {
		return compileStructDecoder(typ)
	}
	if kind == reflect.Pointer {
		return compilePointerDecoder(typ)
	}
	return decodeUnsupported
}

func decodeUnmarshaler(head byte, val reflect.Value, dec *Decoder) error {
	unm, ok := val.Addr().Interface().(Unmarshaler)
	if !ok {
		return nil // the interface was nil.
	}
	dec.opt |= optKeep
	off := dec.pos - 1 // include the head.
	err := dec.skip(head)
	if err != nil {
		return err
	}
	dec.opt &^= optKeep
	return unm.UnmarshalJSON(dec.buf[off:dec.pos])
}

func decodeTextUnmarshaler(head byte, val reflect.Value, dec *Decoder) error {
	const ull = "ull"
	if head == 'n' {
		part, err := dec.readn(len(ull))
		if err == nil && string(part) != ull {
			err = dec.buildErrSyntax(head, ull, part)
		}
		return err
	}
	if head != '"' {
		return dec.errUnmarshalType(head, val.Type())
	}
	unm, ok := val.Addr().Interface().(encoding.TextUnmarshaler)
	if !ok {
		return nil
	}
	slice, err := dec.readString()
	if err != nil {
		return err
	}
	return unm.UnmarshalText(slice)
}

func decodeNumber(head byte, val reflect.Value, dec *Decoder) error {
	const ull = "ull"
	if head == 'n' {
		part, err := dec.readn(len(ull))
		if err == nil && string(part) != ull {
			err = dec.buildErrSyntax(head, ull, part)
		}
		return err
	}
	dec.opt |= optKeep
	off := dec.pos - 1
	if head-'0' >= 10 && head != '-' {
		// special case, don't use dec.errUnmarshalType.
		// see test cases #148 - #151 that used to fail.
		err := dec.skip(head)
		if err != nil {
			return err
		}
		dec.opt &^= optKeep
		src := dec.buf[off:dec.pos]
		return dec.errSyntax("invalid number literal, trying to unmarshal " + strconv.Quote(string(src)) + " into Number")
	}
	err := dec.eatNumber() // needed for syntax check.
	if err != nil {
		return err
	}
	dec.opt &^= optKeep
	src := dec.buf[off:dec.pos]
	val.SetString(string(src)) // make sure to copy
	return nil
}

func decodeString(head byte, val reflect.Value, dec *Decoder) error {
	const ull = "ull"
	if head == 'n' {
		part, err := dec.readn(len(ull))
		if err == nil && string(part) != ull {
			err = dec.buildErrSyntax(head, ull, part)
		}
		return err
	}
	if head != '"' {
		return dec.errUnmarshalType(head, val.Type())
	}
	str, err := dec.readString()
	if err != nil {
		return err
	}
	val.SetString(string(str))
	return nil
}

func decodeInt(head byte, val reflect.Value, dec *Decoder) error {
	const ull = "ull"
	if head == 'n' {
		part, err := dec.readn(len(ull))
		if err == nil && string(part) != ull {
			err = dec.buildErrSyntax(head, ull, part)
		}
		return err
	}
	if head-'0' >= 10 && head != '-' {
		return dec.errUnmarshalType(head, val.Type())
	}
	i64, err := dec.readInt(head)
	if err != nil {
		return err
	}
	if val.OverflowInt(i64) {
		return &UnmarshalTypeError{Value: strconv.FormatInt(i64, 10), Type: val.Type()}
	}
	val.SetInt(i64)
	return nil
}

func decodeUint(head byte, val reflect.Value, dec *Decoder) error {
	const ull = "ull"
	if head == 'n' {
		part, err := dec.readn(len(ull))
		if err == nil && string(part) != ull {
			err = dec.buildErrSyntax(head, ull, part)
		}
		return err
	}
	if head-'0' >= 10 {
		return dec.errUnmarshalType(head, val.Type())
	}
	u64, err := dec.readUint()
	if err != nil {
		return err
	}
	if val.OverflowUint(u64) {
		return &UnmarshalTypeError{Value: strconv.FormatUint(u64, 10), Type: val.Type()}
	}
	val.SetUint(u64)
	return nil
}

func decodeFloat(head byte, val reflect.Value, dec *Decoder) error {
	const ull = "ull"
	if head == 'n' {
		part, err := dec.readn(len(ull))
		if err == nil && string(part) != ull {
			err = dec.buildErrSyntax(head, ull, part)
		}
		return err
	}
	if head-'0' >= 10 && head != '-' {
		return dec.errUnmarshalType(head, val.Type())
	}
	dec.opt |= optKeep
	off := dec.pos
	f64, err := dec.readFloat()
	if err == strconv.ErrRange {
		// rare, slow path.
		dec.pos = off
		err = dec.eatNumber()
		if err != nil {
			return err
		}
		f64, err = strconv.ParseFloat(string(dec.buf[off-1:dec.pos]), 64)
	}
	if err != nil {
		return err
	}
	dec.opt &^= optKeep
	if val.OverflowFloat(f64) {
		return &UnmarshalTypeError{Value: strconv.FormatFloat(f64, 'g', -1, 64), Type: val.Type()}
	}
	val.SetFloat(f64)
	return nil
}

func decodeBool(head byte, val reflect.Value, dec *Decoder) error {
	word := keywords[head]
	if len(word) <= 0 {
		return dec.errUnmarshalType(head, val.Type())
	}
	part, err := dec.readn(len(word))
	if err != nil {
		return err
	}
	if string(part) != word {
		return dec.buildErrSyntax(head, word, part)
	}
	if head != 'n' {
		val.SetBool(head == 't')
	}
	return nil
}

func decodeInterface(head byte, val reflect.Value, dec *Decoder) error {
	const ull = "ull"
	if !val.IsNil() && val.Elem().Kind() == reflect.Pointer && val != val.Elem().Elem() {
		next := val.Elem()        // known to be a pointer
		typ := next.Type().Elem() // a type the pointer above points to.
		is := next.IsNil()
		if is {
			val.Set(reflect.New(typ))
			next = val.Elem()
		}
		if (is || typ.Kind() != reflect.Pointer) && head == 'n' {
			part, err := dec.readn(len(ull))
			if err != nil {
				return err
			}
			if string(part) != ull {
				return dec.buildErrSyntax(head, ull, part)
			}
			val.SetZero()
			return nil
		}
		fnc, ok := decs.get(typ)
		if !ok {
			fnc = compileDecoder(typ)
			decs.set(typ, fnc)
		}
		return fnc(head, next.Elem(), dec)
	}
	if val.NumMethod() != 0 {
		return dec.errUnmarshalType(head, val.Type())
	}
	an, err := dec.readAny(head)
	if err != nil {
		return err
	}
	if an != nil {
		val.Set(reflect.ValueOf(an))
	} else {
		val.SetZero()
	}
	return nil
}

func decodeUnsupported(head byte, val reflect.Value, dec *Decoder) error {
	return errors.New("sonnet: unsupported type: " + val.Type().String())
}
