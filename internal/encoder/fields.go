package encoder

import (
	"reflect"
	"sort"
	"strings"
	"unicode"
	"unsafe"

	"github.com/sugawarayuuta/sonnet/internal/types"
)

const (
	back structError = "bug: this is not an error"
)

type (
	structType struct {
		fields []structField
		typ    *types.Type
	}
	structField struct {
		fun       codec
		ok        bool
		offset    uintptr
		tagged    bool
		omitempty bool
		stringify bool
		json      []byte
		html      []byte
		name      string
		typ       *types.Type
		idx       int
	}
	embeddedField struct {
		index    int
		offset   uintptr
		exported bool
		points   bool
		subtype  *structType
		subfield structField
	}
	structError string
)

func (err structError) Error() string {
	return string(err)
}

func makeStructType(typ *types.Type, prev map[*types.Type]*structType) *structType {

	// prev exists so that it prevents inifite loading.
	if str, ok := prev[typ]; ok {
		return str
	}

	str := &structType{typ: typ}

	prev[typ] = str

	names := make(map[string]struct{})
	embedded := make([]embeddedField, 0, 10)

	ptr := (*types.StructType)(unsafe.Pointer(typ))
	fields := make([]structField, 0, len(ptr.Fields))

	for idx := range ptr.Fields {
		cur := ptr.Fields[idx]

		title, tag := cur.TitleAndTag()

		var (
			name      = title
			exported  = (*cur.Name[0])&(1<<0) != 0
			anonymous = (*cur.Name[0])&(1<<3) != 0
			tagged    bool
			omitempty bool
			stringify bool
		)

		if !exported && !anonymous { 
			continue
		}

		parts := strings.Split(tag.Get("json"), ",")

		if len(parts) != 0 {
			if len(parts[0]) != 0 {
				name, tagged = parts[0], true
			}

			if name == "-" && len(parts) == 1 { 
				continue
			}

			if !isValidTag(name) {
				name = title
			}

			parts = parts[1:]

			for idx := range parts {
				switch parts[idx] {
				case "omitempty":
					omitempty = true
				case "string":
					stringify = true
				}
			}
		}

		if anonymous && !tagged { 

			typ := cur.Type
			kind := reflect.Kind(typ.Kind & types.KindMask)
			points := kind == reflect.Pointer
			if kind == reflect.Pointer {
				_, typ = typ.KeyAndElem()
				kind = reflect.Kind(typ.Kind & types.KindMask)
			}

			if kind == reflect.Struct {
				subtype := makeStructType(typ, prev)

				for subidx := range subtype.fields {
					embedded = append(embedded, embeddedField{
						index:    idx<<32 | subidx,
						offset:   cur.Offset,
						exported: exported,
						subtype:  subtype,
						points:   points,
						subfield: subtype.fields[subidx],
					})
				}
				continue
			}

			if !exported { 
				continue
			}
		}

		fun, ok := load(cur.Type)

		if stringify {
			typ := cur.Type

			kind := reflect.Kind(typ.Kind & types.KindMask)
			if kind == reflect.Pointer {
				_, typ = typ.KeyAndElem()
				kind = reflect.Kind(typ.Kind & types.KindMask)
			}

			if (kind < reflect.Bool || kind > reflect.Float64) && kind != reflect.String {
				stringify = false
			}
		}

		fields = append(fields, structField{
			fun:       fun,
			ok:        ok,
			offset:    cur.Offset,
			tagged:    tagged,
			omitempty: omitempty,
			stringify: stringify,
			name:      name,
			idx:       idx << 32,
			typ:       cur.Type,
		})

		names[name] = struct{}{}
	}

	ambiguousNames := make(map[string]int)
	ambiguousTags := make(map[string]int)

	for name := range names {
		ambiguousNames[name]++
		ambiguousTags[name]++
	}

	for idx := range embedded {
		emb := embedded[idx]
		ambiguousNames[emb.subfield.name]++
		if emb.subfield.tagged {
			ambiguousTags[emb.subfield.name]++
		}
	}

	for idx := range embedded {
		emb := embedded[idx]
		subfield := emb.subfield

		if ambiguousNames[subfield.name] > 1 && !(subfield.tagged && ambiguousTags[subfield.name] == 1) {
			continue // ambiguous embedded field
		}

		if emb.points {
			fun := subfield.fun
			offset := subfield.offset
			subfield.fun = func(dst []byte, ptr unsafe.Pointer, sess Session) ([]byte, error) {
				ptr = *(*unsafe.Pointer)(ptr)
				if ptr == nil {
					return nil, back
				}
				return fun(dst, unsafe.Add(ptr, offset), sess)
			}
			subfield.offset = emb.offset
		} else {
			subfield.offset += emb.offset
		}

		subfield.tagged = false

		subfield.idx = emb.index

		fields = append(fields, subfield)
	}

	for idx := range fields {
		field := &fields[idx]
		length := len(field.name)
		field.json = make([]byte, 0, length+4)
		field.json = append(field.json, ',')
		field.json = appendEscaped(field.json, field.name, false)
		field.json = append(field.json, ':')

		field.html = make([]byte, 0, length+8)
		field.html = append(field.html, ',')
		field.html = appendEscaped(field.html, field.name, true)
		field.html = append(field.html, ':')
	}

	sort.Slice(fields, func(one, two int) bool {
		return fields[one].idx < fields[two].idx
	})

	str.fields = fields

	return str

}

func isValidTag(str string) bool {
	if str == "" {
		return false
	}
	runs := []rune(str)
	for idx := range runs {
		char := runs[idx]
		switch {
		case strings.ContainsRune("!#$%&()*+-./:;<=>?@[]^_{|}~ ", char):
			// Backslash and quote chars are reserved, but
			// otherwise any punctuation chars are allowed
			// in a tag name.
		default:
			if !unicode.IsLetter(char) && !unicode.IsDigit(char) {
				return false
			}
		}
	}
	return true
}
