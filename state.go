package sonnet

import (
	"io"
)

func stateValue(self *Decoder) ([]uint8, error) {

	bytes := self.scanner.scan()
	if len(bytes) == 0 {
		return nil, io.ErrUnexpectedEOF
	}

	switch bytes[0] {
	case '{':
		self.state = stateObjectKey
		self.append(true)
	case '[':
		self.state = stateArrayValue
		self.append(false)
	default:
		self.state = stateEOF
	}
	return bytes, nil
}

func stateObjectKey(self *Decoder) ([]uint8, error) {

	bytes := self.scanner.scan()
	if len(bytes) == 0 {
		return nil, io.ErrUnexpectedEOF
	}

	switch bytes[0] {
	case '}':
		if self.len() == 1 {
			self.state = stateEOF
			return bytes, nil
		}
		if self.tail() {
			self.state = stateObjectComma
			return bytes, nil
		}
		self.state = stateArrayComma
		return bytes, nil
	case '"':
		self.state = stateObjectColon
		return bytes, nil
	}
	return nil, &SyntaxError{
		msg:    "missing string object key, got: " + string(bytes),
		Offset: self.InputOffset(),
	}
}

func stateObjectColon(self *Decoder) ([]uint8, error) {

	bytes := self.scanner.scan()
	if len(bytes) == 0 {
		return nil, io.ErrUnexpectedEOF
	}

	switch bytes[0] {
	case ':':
		return stateObjectValue(self)
	}

	return nil, &SyntaxError{
		msg:    "missing colon, got: " + string(bytes),
		Offset: self.InputOffset(),
	}
}

func stateObjectValue(self *Decoder) ([]uint8, error) {

	bytes := self.scanner.scan()
	if len(bytes) == 0 {
		return nil, io.ErrUnexpectedEOF
	}
	switch bytes[0] {
	case '{':
		self.state = stateObjectKey
		self.append(true)
	case '[':
		self.state = stateArrayValue
		self.append(false)
	default:
		self.state = stateObjectComma
	}
	return bytes, nil
}

func stateObjectComma(self *Decoder) ([]uint8, error) {

	bytes := self.scanner.scan()
	if len(bytes) == 0 {
		return nil, io.ErrUnexpectedEOF
	}
	switch bytes[0] {
	case '}':
		if self.len() == 1 {
			self.state = stateEOF
			return bytes, nil
		}
		if self.tail() {
			self.state = stateObjectComma
			return bytes, nil
		}
		self.state = stateArrayComma
		return bytes, nil
	case ',':
		return stateObjectKey(self)
	}
	return nil, &SyntaxError{
		msg:    "missing comma, got: " + string(bytes),
		Offset: self.InputOffset(),
	}
}

func stateArrayValue(self *Decoder) ([]uint8, error) {

	bytes := self.scanner.scan()
	if len(bytes) == 0 {
		return nil, io.ErrUnexpectedEOF
	}

	switch bytes[0] {
	case '{':
		self.state = stateObjectKey
		self.append(true)
	case '[':
		self.state = stateArrayValue
		self.append(false)
	case ']':
		if self.len() == 1 {
			self.state = stateEOF
			return bytes, nil
		} else if self.tail() {
			self.state = stateObjectComma
			return bytes, nil
		}
		self.state = stateArrayComma
		return bytes, nil
	default:
		self.state = stateArrayComma
	}
	return bytes, nil
}

func stateArrayComma(self *Decoder) ([]uint8, error) {

	bytes := self.scanner.scan()
	if len(bytes) == 0 {
		return nil, io.ErrUnexpectedEOF
	}
	switch bytes[0] {
	case ']':
		if self.len() == 1 {
			self.state = stateEOF
			return bytes, nil
		}
		if self.tail() {
			self.state = stateObjectComma
			return bytes, nil
		}
		self.state = stateArrayComma
		return bytes, nil
	case ',':
		return stateArrayValue(self)
	}
	return nil, &SyntaxError{
		msg:    "missing comma, got: " + string(bytes),
		Offset: self.InputOffset(),
	}
}

func stateEOF(self *Decoder) ([]uint8, error) {
	return nil, io.EOF
}
