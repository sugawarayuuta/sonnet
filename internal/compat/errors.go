package compat

import (
	"reflect"
)

var (
	vals = [256]string{
		't': `t (head): "true"?`,
		'f': `f (head): "false"?`,
		'n': `n (head): "null"?`,
		'1': `1 (head): number?`,
		'2': `2 (head): number?`,
		'3': `3 (head): number?`,
		'4': `4 (head): number?`,
		'5': `5 (head): number?`,
		'6': `6 (head): number?`,
		'7': `7 (head): number?`,
		'8': `8 (head): number?`,
		'9': `9 (head): number?`,
		'0': `0 (head): number?`,
		'-': `- (head): number?`,
		'"': `" (head): string?`,
		'[': `[ (head): array?`,
		'{': `{ (head): object?`,
	}
)

func NewInvalidUnmarshalError(typ reflect.Type) *InvalidUnmarshalError {
	return &InvalidUnmarshalError{Type: typ}
}

// An InvalidUnmarshalError describes an invalid argument passed to Unmarshal.
// (The argument to Unmarshal must be a non-nil pointer.)
type InvalidUnmarshalError struct {
	Type reflect.Type
}

func (err *InvalidUnmarshalError) Error() string {
	if err.Type == nil {
		return "sonnet: Unmarshal(nil)"
	}

	if err.Type.Kind() != reflect.Pointer {
		return "sonnet: Unmarshal(non-pointer " + err.Type.String() + ")"
	}
	return "sonnet: Unmarshal(nil " + err.Type.String() + ")"
}

func NewUnmarshalTypeError(head byte, typ reflect.Type, offset int64) *UnmarshalTypeError {
	val := vals[head]
	if val == "" {
		val = string(head) + ` (head): unknown`
	}
	return &UnmarshalTypeError{Value: val, Type: typ, Offset: offset}
}

// An UnmarshalTypeError describes a JSON value that was
// not appropriate for a value of a specific Go type.
type UnmarshalTypeError struct {
	Value  string       // description of JSON value - "bool", "array", "number -5"
	Type   reflect.Type // type of Go value it could not be assigned to
	Offset int64        // error occurred after reading Offset bytes
	Struct string       // name of the struct type containing the field
	Field  string       // the full path from root node to the field
}

func (err *UnmarshalTypeError) Error() string {
	if err.Struct != "" || err.Field != "" {
		return "sonnet: cannot unmarshal " + err.Value + " into Go struct field " + err.Struct + "." + err.Field + " of type " + err.Type.String()
	}
	return "sonnet: cannot unmarshal " + err.Value + " into Go value of type " + err.Type.String()
}

func NewUnsupportedValueError(val reflect.Value, str string) *UnsupportedValueError {
	return &UnsupportedValueError{Value: val, Str: str}
}

// An UnsupportedValueError is returned by Marshal when attempting
// to encode an unsupported value.
type UnsupportedValueError struct {
	Value reflect.Value
	Str   string
}

func (err *UnsupportedValueError) Error() string {
	return "sonnet: unsupported value: " + err.Str
}

func NewUnsupportedTypeError(typ reflect.Type) *UnsupportedTypeError {
	return &UnsupportedTypeError{Type: typ}
}

// An UnsupportedTypeError is returned by Marshal when attempting
// to encode an unsupported value type.
type UnsupportedTypeError struct {
	Type reflect.Type
}

func (err *UnsupportedTypeError) Error() string {
	return "sonnet: unsupported type: " + err.Type.String()
}

func NewSyntaxError(msg string, offset int64) *SyntaxError {
	return &SyntaxError{msg: msg, Offset: offset}
}

// A SyntaxError is a description of a JSON syntax error.
// Unmarshal will return a SyntaxError if the JSON can't be parsed.
type SyntaxError struct {
	msg    string // description of error
	Offset int64  // error occurred after reading Offset bytes
}

// just returns the msg. i may add more context later.
func (err *SyntaxError) Error() string {
	return err.msg
}

func NewMarshalerError(typ reflect.Type, err error, src string) *MarshalerError {
	return &MarshalerError{Type: typ, Err: err, sourceFunc: src}
}

// A MarshalerError represents an error from calling a MarshalJSON or MarshalText method.
type MarshalerError struct {
	Type       reflect.Type
	Err        error
	sourceFunc string
}

func (err *MarshalerError) Error() string {
	sourceFunc := err.sourceFunc
	if sourceFunc == "" {
		sourceFunc = "MarshalJSON"
	}
	return "sonnet: error calling " + sourceFunc + " for type " + err.Type.String() + ": " + err.Err.Error()
}

// Unwrap returns the underlying error.
func (err *MarshalerError) Unwrap() error {
	return err.Err
}
