package sonnet

import (
	"reflect"
	"strings"
	"unsafe"
)

// structType represents a content for struct types.
// it is used to get offsets/types of struct fields, see below.
type structType struct {
	rtype
	pkgPath [1]*byte
	fields  []structField
}

// structField is a field in a struct.
// it contains its name, type, offset.
type structField struct {
	name   [1]*byte
	typ    *rtype
	offset uintptr
}

// stringHeader represents reflect.StringHeader but with unsafe.Pointer
// instead of uintptr in the first field
type stringHeader struct {
	ptr unsafe.Pointer
	len int
}

// sliceHeader represents reflect.SliceHeader but with unsafe.Pointer
// instead of uintptr in the first field
type sliceHeader struct {
	ptr unsafe.Pointer
	len int
	cap int
}

// (*rtype).grow grows a slice that is in typ type.
// new cap will be (previousCap + 1) * 2 .
func (typ *rtype) grow(header *sliceHeader) {
	cap := (header.cap + 1) * 2

	newHeader := sliceHeader{
		ptr: typ.unsafe_NewArray(cap),
		len: header.len,
		cap: cap,
	}

	if header.cap > 0 {
		header.len = typ.typedslicecopy(newHeader, *header)
	}

	header.ptr = newHeader.ptr
	header.cap = cap
}

// (*rtype).fields gives store containing struct offset/type.
// to know more about store, see store.go.
func (typ *rtype) fields() store {
	str := (*structType)(unsafe.Pointer(typ))
	var caches store
	caches.init(uint64(len(str.fields)))
	// init the caches with length of fields, equivalent would be NumField.

	for idx := range str.fields {
		title, tag := str.fields[idx].key()

		var name string
		var ok bool
		if name, ok = tag.Lookup("json"); ok {
			if name == "-" {
				continue
			}
			idx := strings.IndexByte(name, ',')
			if idx != -1 {
				name = name[:idx]
			}
			// so that it wouldn't break even if people set something like
			// "omitempty" as an option.
		} else {
			name = strings.ToLower(title)
		}
		caches.set(name, str.fields[idx])
	}

	return caches
}

// (structField).key parses struct tag and field name in a struct.
// similar func is in the standard library.
func (structField structField) key() (string, reflect.StructTag) {

	if structField.name[0] == nil {
		return "", ""
	}

	one, two := structField.readVarint(1)
	title := *(*string)(unsafe.Pointer(&stringHeader{
		ptr: unsafe.Add(unsafe.Pointer(structField.name[0]), one+1),
		len: two,
	}))

	if (*structField.name[0])&(1<<1) == 0 {
		return title, ""
	}

	three, four := structField.readVarint(one + two + 1)
	tag := *(*reflect.StructTag)(unsafe.Pointer(&stringHeader{
		ptr: unsafe.Add(unsafe.Pointer(structField.name[0]), one+two+three+1),
		len: four,
	}))
	// cast to reflect.StructTag, not string.
	// only difference is that you can call Get/Lookup method from it.

	return title, tag
}

// readVarint parses a varint as encoded by encoding/binary.
// It returns the number of encoded bytes and the encoded value.
func (structField structField) readVarint(off int) (int, int) {
	val := 0
	for idx := 0; ; idx++ {
		u8 := *(*byte)(unsafe.Add(unsafe.Pointer(structField.name[0]), off+idx))
		val += int(u8&0x7f) << (7 * idx)
		if u8&0x80 == 0 {
			return idx + 1, val
		}
	}
}

// toString converts byte slices to a string, using unsafe.
// this should be replaced after go releases new unsafe features.
func toString(bytes []byte) string {
	return *(*string)(unsafe.Pointer(&bytes))
}
