package sonnet

import "encoding/json"

type InvalidUnmarshalError = json.InvalidUnmarshalError

type UnmarshalTypeError = json.UnmarshalTypeError

// doesn't use type alias because the field msg isn't exported.
type SyntaxError struct {
	msg string
	Offset int64
}

// just returns the msg. i may add more context later.
func (syntaxError *SyntaxError) Error() string {
	return syntaxError.msg
}
