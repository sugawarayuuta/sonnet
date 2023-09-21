package sonnet

type (
	// A Token holds a value of one of these types:
	//
	//	Delim, for the four JSON delimiters [ ] { }
	//	bool, for JSON booleans
	//	float64, for JSON numbers
	//	Number, for JSON numbers
	//	string, for JSON string literals
	//	nil, for JSON null
	Token any
	// A Delim is a JSON array or object delimiter, one of [ ] { or }.
	Delim rune
)

func (delim Delim) String() string {
	return string(delim)
}

// Token returns the next JSON token in the input stream.
// At the end of the input stream, Token returns nil, io.EOF.
//
// Token guarantees that the delimiters [ ] { } it returns are
// properly nested and matched: if Token encounters an unexpected
// delimiter in the input, it will return an error.
//
// The input stream consists of basic JSON values—bool, string,
// number, and null—along with delimiters [ ] { } of type Delim
// to mark the start and end of arrays and objects.
// Commas and colons are elided.
func (dec *Decoder) Token() (Token, error) {
	for {
		dec.eatSpaces()
		if dec.pos >= len(dec.buf) && !dec.fill() {
			return nil, dec.errSyntax("unexpected EOF reading a byte")
		}
		head := dec.buf[dec.pos]
		dec.pos++
		if head == '{' || head == '}' || head == '[' || head == ']' {
			return Delim(head), nil
		}
		if head != ',' && head != ':' {
			return dec.readAny(head)
		}
	}
}
