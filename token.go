package sonnet

import "encoding/json"

type Token = json.Token

type Delim = json.Delim

func (decoder *Decoder) Token() (Token, error) {
	bytes, err := decoder.state(decoder)
	if err != nil {
		return nil, err
	}
	switch bytes[0] {
	case '{', '}', '[', ']':
		return Delim(bytes[0]), nil
	case 't', 'f':
		return bytes[0] == 't', nil
	case 'n':
		return nil, nil
	case '"':
		return string(bytes[1 : len(bytes)-1]), nil
	default:
		return f64(bytes), nil
	}
}

type Number = json.Number
