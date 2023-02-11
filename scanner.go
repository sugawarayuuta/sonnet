package sonnet

type scanner struct {
	reader reader
	pos    int
}

var (
	isSpace = [256]bool{
		' ':  true,
		'\n': true,
		'\t': true,
		'\r': true,
	}
	isSign = [256]bool{
		'{': true,
		'}': true,
		'[': true,
		']': true,
		',': true,
		':': true,
	}
	isDigit = [256]bool{
		'0': true,
		'1': true,
		'2': true,
		'3': true,
		'4': true,
		'5': true,
		'6': true,
		'7': true,
		'8': true,
		'9': true,
		'-': true,
		'e': true,
		'E': true,
		'.': true,
	}
)

func (scanner *scanner) scan() []byte {
	scanner.reader.pos += scanner.pos
	bytes := scanner.reader.bytes[scanner.reader.pos:]

	for {
		for idx := range bytes {

			if isSpace[bytes[idx]] {
				continue
			} else if isSign[bytes[idx]] {
				scanner.pos = idx + 1
				return bytes[idx:scanner.pos]
			}

			scanner.reader.pos += idx

			switch bytes[idx] {
			case '"':
				scanner.pos = scanner.scanString()
			case 't':
				scanner.pos = scanner.scanThis("true")
			case 'f':
				scanner.pos = scanner.scanThis("false")
			case 'n':
				scanner.pos = scanner.scanThis("null")
			default:
				scanner.pos = scanner.scanNumber()
			}
			return scanner.reader.bytes[scanner.reader.pos:][:scanner.pos]
		}

		scanner.reader.pos += len(bytes)

		if scanner.reader.more() == 0 {
			return nil
		}

		bytes = scanner.reader.bytes[scanner.reader.pos:]
	}
}

func (scanner *scanner) scanString() int {
	bytes := scanner.reader.bytes[scanner.reader.pos+1:]
	var pos int
	var prev byte
	for {
		for idx := range bytes {
			pos++
			if bytes[idx] == '"' && prev != '\\' {
				return pos + 1
			}
			prev = bytes[idx]
		}

		if scanner.reader.more() == 0 {
			return 0
		}
		bytes = scanner.reader.bytes[scanner.reader.pos+pos+1:]
	}
}

func (scanner *scanner) scanNumber() int {
	bytes := scanner.reader.bytes[scanner.reader.pos:]
	var pos int
	var cur byte
	for {
		for idx := range bytes {
			cur = bytes[idx]
			if !isDigit[cur] {
				return pos
			}
			pos++
		}

		if scanner.reader.more() == 0 {
			if cur >= '0' && cur <= '9' {
				return pos
			}
			return 0
		}
		bytes = scanner.reader.bytes[scanner.reader.pos+pos:]
	}
}

func (scanner *scanner) scanThis(this string) int {
	for {
		bytes := scanner.reader.bytes[scanner.reader.pos:]
		length := len(this)

		if len(bytes) >= length {
			if toString(bytes[:length]) != this {
				return 0
			}
			return length
		}

		if scanner.reader.more() == 0 {
			return 0
		}
	}
}
