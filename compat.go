package sonnet

import (
	"encoding"
	"errors"
	"reflect"
	"strconv"
)

type (
	// RawMessage is a raw encoded JSON value.
	// It implements Marshaler and Unmarshaler and can
	// be used to delay JSON decoding or precompute a JSON encoding.
	RawMessage []byte
	// Marshaler is the interface implemented by types that
	// can marshal themselves into valid JSON.
	Marshaler interface {
		MarshalJSON() ([]byte, error)
	}
	// Unmarshaler is the interface implemented by types
	// that can unmarshal a JSON description of themselves.
	// The input can be assumed to be a valid encoding of
	// a JSON value. UnmarshalJSON must copy the JSON data
	// if it wishes to retain the data after returning.
	//
	// By convention, to approximate the behavior of Unmarshal itself,
	// Unmarshalers implement UnmarshalJSON([]byte("null")) as a no-op.
	Unmarshaler interface {
		UnmarshalJSON([]byte) error
	}
	// A Number represents a JSON number literal.
	Number string
	// An InvalidUnmarshalError describes an invalid argument passed to Unmarshal.
	// (The argument to Unmarshal must be a non-nil pointer.)
	InvalidUnmarshalError struct {
		Type reflect.Type
	}
	// A SyntaxError is a description of a JSON syntax error.
	// Unmarshal will return a SyntaxError if the JSON can't be parsed.
	SyntaxError struct {
		msg    string // description of error
		Offset int64  // error occurred after reading Offset bytes
	}
	// An UnmarshalTypeError describes a JSON value that was
	// not appropriate for a value of a specific Go type.
	UnmarshalTypeError struct {
		Value  string       // description of JSON value - "bool", "array", "number -5"
		Type   reflect.Type // type of Go value it could not be assigned to
		Offset int64        // error occurred after reading Offset bytes
		Struct string       // name of the struct type containing the field
		Field  string       // the full path from root node to the field
	}
	// An UnsupportedValueError is returned by Marshal when attempting
	// to encode an unsupported value.
	UnsupportedValueError struct {
		Value reflect.Value
		Str   string
	}
	// An UnsupportedTypeError is returned by Marshal when attempting
	// to encode an unsupported value type.
	UnsupportedTypeError struct {
		Type reflect.Type
	}
	// A MarshalerError represents an error from calling a MarshalJSON or MarshalText method.
	MarshalerError struct {
		Type       reflect.Type
		Err        error
		sourceFunc string
	}
)

var (
	marshaler       = reflect.TypeOf((*Marshaler)(nil)).Elem()
	unmarshaler     = reflect.TypeOf((*Unmarshaler)(nil)).Elem()
	textMarshaler   = reflect.TypeOf((*encoding.TextMarshaler)(nil)).Elem()
	textUnmarshaler = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()
	number          = reflect.TypeOf(Number(""))
)

// MarshalJSON returns msg as the JSON encoding of msg.
func (msg RawMessage) MarshalJSON() ([]byte, error) {
	if msg == nil {
		return []byte("null"), nil
	}
	return msg, nil
}

// UnmarshalJSON sets *msg to a copy of data.
func (msg *RawMessage) UnmarshalJSON(data []byte) error {
	if msg == nil {
		return errors.New("json.RawMessage: UnmarshalJSON on nil pointer")
	}
	*msg = append((*msg)[0:0], data...)
	return nil
}

var _ Marshaler = (*RawMessage)(nil)
var _ Unmarshaler = (*RawMessage)(nil)

// String returns the literal text of the number.
func (num Number) String() string {
	return string(num)
}

// Float64 returns the number as a float64.
func (num Number) Float64() (float64, error) {
	return strconv.ParseFloat(string(num), 64)
}

// Int64 returns the number as an int64.
func (num Number) Int64() (int64, error) {
	return strconv.ParseInt(string(num), 10, 64)
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

func (err *SyntaxError) Error() string {
	return "sonnet: " + err.msg
}

func (err *UnmarshalTypeError) Error() string {
	if err.Struct != "" || err.Field != "" {
		return "sonnet: cannot unmarshal " + err.Value + " into Go struct field " + err.Struct + "." + err.Field + " of type " + err.Type.String()
	}
	return "sonnet: cannot unmarshal " + err.Value + " into Go value of type " + err.Type.String()
}

func (err *UnsupportedValueError) Error() string {
	return "sonnet: unsupported value: " + err.Str
}

func (err *UnsupportedTypeError) Error() string {
	return "sonnet: unsupported type: " + err.Type.String()
}

func (err *MarshalerError) Error() string {
	srcFunc := err.sourceFunc
	if srcFunc == "" {
		srcFunc = "MarshalJSON"
	}
	return "sonnet: error calling " + srcFunc + " for type " + err.Type.String() + ": " + err.Err.Error()
}
