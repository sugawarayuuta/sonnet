package sonnet

import (
	"github.com/sugawarayuuta/sonnet/internal/compat"
)

type (
	// Marshaler is the interface implemented by types that
	// can marshal themselves into valid JSON.
	Marshaler = compat.Marshaler
	// Unmarshaler is the interface implemented by types
	// that can unmarshal a JSON description of themselves.
	// The input can be assumed to be a valid encoding of
	// a JSON value. UnmarshalJSON must copy the JSON data
	// if it wishes to retain the data after returning.
	//
	// By convention, to approximate the behavior of Unmarshal itself,
	// Unmarshalers implement UnmarshalJSON([]byte("null")) as a no-op.
	Unmarshaler = compat.Unmarshaler
	// RawMessage is a raw encoded JSON value.
	// It implements Marshaler and Unmarshaler and can
	// be used to delay JSON decoding or precompute a JSON encoding.
	RawMessage = compat.RawMessage
)

type (
	// A Token holds a value of one of these types:
	//
	//	Delim, for the four JSON delimiters [ ] { }
	//	bool, for JSON booleans
	//	float64, for JSON numbers
	//	Number, for JSON numbers
	//	string, for JSON string literals
	//	nil, for JSON null
	Token = compat.Token
	// A Delim is a JSON array or object delimiter, one of [ ] { or }.
	Delim = compat.Delim
	// A Number represents a JSON number literal.
	Number = compat.Number
)

type (
	// An InvalidUnmarshalError describes an invalid argument passed to Unmarshal.
	// (The argument to Unmarshal must be a non-nil pointer.)
	InvalidUnmarshalError = compat.InvalidUnmarshalError
	// A MarshalerError represents an error from calling a MarshalJSON or MarshalText method.
	MarshalerError = compat.MarshalerError
	// A SyntaxError is a description of a JSON syntax error.
	// Unmarshal will return a SyntaxError if the JSON can't be parsed.
	SyntaxError = compat.SyntaxError
	// An UnmarshalTypeError describes a JSON value that was
	// not appropriate for a value of a specific Go type.
	UnmarshalTypeError = compat.UnmarshalTypeError
	// An UnsupportedValueError is returned by Marshal when attempting
	// to encode an unsupported value.
	UnsupportedValueError = compat.UnsupportedValueError
	// An UnsupportedTypeError is returned by Marshal when attempting
	// to encode an unsupported value type.
	UnsupportedTypeError = compat.UnsupportedTypeError
)
