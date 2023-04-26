package encoder

import "unsafe"

type (
	Session struct {
		// if true, escapes html strings.
		html bool
		// doing the same thing as encoding/json. see below for more.
		// Keep track of what pointers we've seen in the current recursive call
		// path, to avoid cycles that could lead to a stack overflow. Only do
		// the relatively expensive map operations if ptrLevel is larger than
		// startDetectingCyclesAfter, so that we skip the work if we're within a
		// reasonable amount of nested pointers deep.
		ptrLevel uint
		// encoding/json uses any for map key because they want to avoid
		// unsafe package. this is not a thing here, so we'll just use it
		ptrSeen map[unsafe.Pointer]struct{}
	}
)

const (
	startDetectingCyclesAfter = 1000
)

func NewSession(html bool) Session {
	sess := Session{
		html:    html,
		// don't initialize ptrSeen; it cost 7% on small structs 
		// instead, do it when ptrLevel > startDetectingCyclesAfter.
	}
	return sess
}
