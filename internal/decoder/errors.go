package decoder

type (
	typeError  string
	fieldError string
	rangeError string
)

func (err typeError) Error() string {
	return "sonnet: unsupported type: " + string(err)
}

func (err fieldError) Error() string {
	return "sonnet: unknown field: " + string(err)
}

func (err rangeError) Error() string {
	return "sonnet: number literal created an overflow: " + string(err)
}
