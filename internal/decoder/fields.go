package decoder

import (
	"errors"
	"reflect"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
	"unsafe"

	"github.com/sugawarayuuta/sonnet/internal/types"
)

type (
	structType struct {
		fields        []structField
		fieldsTab     table
		fieldCaseTab  table
		typ           *types.Type
		duplicate     bool
		ascii         bool
	}
	structField struct {
		fun       codec
		ok        bool
		offset    uintptr
		tagged    bool
		stringify bool
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
)

var (
	toLower [256]byte
)

func init() {
	for idx := 0; idx < 256; idx++ {
		toLower[idx] = byte(idx)
		if idx >= 'A' && idx <= 'Z' {
			toLower[idx] = byte(idx | ' ')
		}
	}
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

	size := len(ptr.Fields)
	fields := make([]structField, 0, size)
	str.fieldsTab.prep(uint64(size + 1))
	str.fieldCaseTab.prep(uint64(size + 1))

	for idx := range ptr.Fields {
		cur := ptr.Fields[idx]

		title, tag := cur.TitleAndTag()

		var (
			name      = title
			exported  = (*cur.Name[0])&(1<<0) != 0
			anonymous = (*cur.Name[0])&(1<<3) != 0
			tagged    bool
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
			continue 
		}

		if emb.points {
			fun := subfield.fun
			offset := subfield.offset
			subfield.fun = func(head byte, ptr unsafe.Pointer, sess *Session) error {
				typ := emb.subtype.typ
				temp := (*unsafe.Pointer)(ptr)
				if *temp == nil {
					if !emb.exported {
						return errors.New("sonnet: cannot set embedded pointer to unexported struct: " +
							typ.Reflect().String())
					}
					*temp = typ.New()
				}
				return fun(head, unsafe.Add(*temp, offset), sess)
			}
			subfield.offset = emb.offset
		} else {
			subfield.offset += emb.offset
		}

		subfield.tagged = false

		subfield.idx = emb.index

		fields = append(fields, subfield)
	}

	sort.Slice(fields, func(one, two int) bool {
		return fields[one].idx < fields[two].idx
	})

	str.fields = fields

	ascii := true
	vis := make(map[string]struct{})
	cases := make([]structField, 0, len(str.fields))
	for idx := range str.fields {
		field := str.fields[idx]

		ascii = ascii && isASCII(field.name)
		lower := strings.ToLower(field.name)
		if _, ok := vis[lower]; !ok {
			vis[lower] = struct{}{}
			field.name = lower
			cases = append(cases, field)
			continue
		}

		str.duplicate = true
	}

	str.ascii = ascii
	str.fieldsTab.set(str.fields)
	str.fieldCaseTab.set(cases)

	return str
}

func isValidTag(str string) bool {
	if str == "" {
		return false
	}
	for _, char := range str {
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

func isASCII(str string) bool {
	for idx := range str {
		if str[idx] > unicode.MaxASCII {
			return false
		}
	}
	return true
}

func appendToLower(dst []byte, src []byte) []byte {
	var idx int
	for idx < len(src) {
		run, size := utf8.DecodeRune(src[idx:])
		idx += size
		dst = utf8.AppendRune(dst, unicode.ToLower(run))
	}
	return dst
}
