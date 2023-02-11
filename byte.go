package sonnet

// might be a temporary solution, rewrite this when
// i come up with a better idea
func Unmarshal(bytes []byte, this any) error {
	decoder := Decoder{
		state: stateValue,
		stack: make([]bool, 0, 4),
		scanner: scanner{
			reader: reader{
				bytes: bytes,
			},
		},
	}
	return decoder.Decode(this)
}

// this too. seems like a terrible implementation.
// i don't know if it's the right way to put any there.
func Valid(bytes []byte) bool {
	return Unmarshal(bytes, new(any)) == nil
}
