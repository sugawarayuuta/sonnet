package sonnet

func (decoder *Decoder) tail() bool {
	decoder.stack = decoder.stack[:decoder.len()-1]
	if decoder.len() == 0 {
		return false
	}
	return decoder.stack[decoder.len()-1]
}

func (decoder *Decoder) append(bol bool) {
	decoder.stack = append(decoder.stack, bol)
}

func (decoder *Decoder) len() int {
	return len(decoder.stack)
}
