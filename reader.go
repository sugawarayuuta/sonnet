package sonnet

import (
	"io"
)

type reader struct {
	bytes []byte
	pos   int
	input io.Reader
	err   error
	total int
}

func (reader *reader) more() int {
	// returns if there is an error, 
	// or if it's doesn't have io.Reader
	// for the latter, see byte.go .
	if reader.err != nil || reader.input == nil {
		return 0
	}

	remaining := len(reader.bytes) - reader.pos

	if remaining == 0 {
		reader.bytes = reader.bytes[:0]
		reader.total += reader.pos
		reader.pos = 0
	}

	if cap(reader.bytes)-remaining >= 1024 {
		copy(reader.bytes, reader.bytes[reader.pos:])
		reader.total += reader.pos
		reader.pos = 0
	} else if cap(reader.bytes)-len(reader.bytes) < 1024 {
		length := cap(reader.bytes) * 2
		// resize if too small.
		if length < 1024*4 {
			length = 1024 * 4
		}
		bytes := make([]byte, length)

		copy(bytes, reader.bytes[reader.pos:])
		reader.bytes = bytes
		reader.total += reader.pos
		reader.pos = 0
	}

	remaining += reader.pos
	read, err := reader.input.Read(reader.bytes[remaining:cap(reader.bytes)])

	reader.err = err
	reader.bytes = reader.bytes[:remaining+read]
	return read
}
