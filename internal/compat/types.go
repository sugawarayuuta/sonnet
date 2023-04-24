package compat

import (
	"encoding"
	"encoding/json"
	"reflect"

	"github.com/sugawarayuuta/sonnet/internal/types"
)

type (
	Marshaler   = json.Marshaler
	Unmarshaler = json.Unmarshaler
	RawMessage  = json.RawMessage
)

type (
	Token  = json.Token
	Delim  = json.Delim
	Number = json.Number
)

var (
	MarshalerType       = reflect.TypeOf((*Marshaler)(nil)).Elem()
	UnmarshalerType     = reflect.TypeOf((*Unmarshaler)(nil)).Elem()
	TextMarshalerType   = reflect.TypeOf((*encoding.TextMarshaler)(nil)).Elem()
	TextUnmarshalerType = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()
)

var (
	NumberType, _ = types.TypeAndPointerOf(Number(""))
)
