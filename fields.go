package sonnet

import (
	"reflect"
	"sort"
	"strings"
	"unicode"
)

type (
	flag   byte
	fields struct {
		flds    []field
		fldsMap *perf
		caseMap *perf
		flg     flag
	}
	field struct {
		dec      decoder
		enc      encoder
		flg      flag
		name     string
		nameJSON []byte
		nameHTML []byte
		typ      reflect.Type
		idxs     []int
	}
	byIdx []field
)

const (
	flagDup = 1 << iota
	flagAZ95
	flagTag
	flagString
	flagOmitempty
)

func (by byIdx) Len() int {
	return len(by)
}

func (by byIdx) Swap(fst, sec int) {
	by[fst], by[sec] = by[sec], by[fst]
}

func (by byIdx) Less(fst, sec int) bool {
	for idx := range by[fst].idxs {
		if idx >= len(by[sec].idxs) {
			return false
		}
		if by[fst].idxs[idx] != by[sec].idxs[idx] {
			return by[fst].idxs[idx] < by[sec].idxs[idx]
		}
	}
	return len(by[fst].idxs) < len(by[sec].idxs)
}

// typeFields returns a list of fields that JSON should recognize for the given type.
// The algorithm is breadth-first search over the set of structs to include - the top struct
// and then any reachable anonymous structs.
func makeFields(typ reflect.Type) fields {
	var flds, currFlds, nextFlds []field
	var currCnt, nextCnt map[reflect.Type]int

	nextFlds = append(nextFlds, field{typ: typ})
	vis := make(map[reflect.Type]struct{})

	for len(nextFlds) > 0 {
		currFlds, nextFlds = nextFlds, currFlds[:0]
		currCnt, nextCnt = nextCnt, map[reflect.Type]int{}

		for _, fld := range currFlds {
			if _, ok := vis[fld.typ]; ok {
				continue
			}
			vis[fld.typ] = struct{}{}

			// Scan fld.typ for fields to include.
			for idx := 0; idx < fld.typ.NumField(); idx++ {
				str := fld.typ.Field(idx)
				if str.Anonymous {
					typ := str.Type
					if typ.Kind() == reflect.Pointer {
						typ = typ.Elem()
					}
					if !str.IsExported() && typ.Kind() != reflect.Struct {
						// Ignore embedded fields of unexported non-struct types.
						continue
					}
					// Do not ignore embedded fields of unexported struct types
					// since they may have exported fields.
				} else if !str.IsExported() {
					// Ignore unexported non-embedded fields.
					continue
				}
				tag := str.Tag.Get("json")
				if tag == "-" {
					continue
				}
				var name string
				spl := strings.Split(tag, ",")
				if len(spl) > 0 {
					name = spl[0]
					spl = spl[1:]
				}
				if !isValidTag(name) {
					name = ""
				}
				idxs := make([]int, len(fld.idxs)+1)
				copy(idxs, fld.idxs)
				idxs[len(fld.idxs)] = idx

				typ := str.Type
				if typ.Name() == "" && typ.Kind() == reflect.Pointer {
					typ = typ.Elem()
				}

				var flg flag
				for idx := range spl {
					if spl[idx] == "string" {
						flg |= flagString
					}
					if spl[idx] == "omitempty" {
						flg |= flagOmitempty
					}
				}

				const accept = reflect.Float64 - reflect.Bool
				if typ.Kind()-reflect.Bool > accept && typ.Kind() != reflect.String {
					flg &^= flagString
				}

				// Record found field and index sequence.
				if name != "" || !str.Anonymous || typ.Kind() != reflect.Struct {
					if name != "" {
						flg |= flagTag
					} else {
						name = str.Name
					}
					app := field{
						name: name,
						idxs: idxs,
						typ:  typ,
						flg:  flg,
					}
					flds = append(flds, app)
					if currCnt[fld.typ] > 1 {
						// If there were multiple instances, add a second,
						// so that the annihilation code will see a duplicate.
						// It only cares about the distinction between 1 or 2,
						// so don't bother generating any more copies.
						flds = append(flds, flds[len(flds)-1])
					}
					continue
				}

				// Record new anonymous struct to explore in nextFlds round.
				nextCnt[typ]++
				if nextCnt[typ] == 1 {
					app := field{
						name: typ.Name(),
						idxs: idxs,
						typ:  typ,
					}
					nextFlds = append(nextFlds, app)
				}
			}
		}
	}

	sort.Slice(flds, func(fst, sec int) bool {
		// sort field by name, breaking ties with depth, then
		// breaking ties with "name came from json tag", then
		// breaking ties with index sequence.
		if flds[fst].name != flds[sec].name {
			return flds[fst].name < flds[sec].name
		}
		if len(flds[fst].idxs) != len(flds[sec].idxs) {
			return len(flds[fst].idxs) < len(flds[sec].idxs)
		}
		if (flds[fst].flg^flds[sec].flg)&flagTag != 0 {
			return flds[fst].flg&flagTag != 0
		}
		return byIdx(flds).Less(fst, sec)
	})

	// Delete all fields that are hidden by the Go rules for embedded fields,
	// except that fields with JSON tags are promoted.

	// The fields are sorted in primary order of name, secondary order
	// of field index length. Loop over names; for each name, delete
	// hidden fields by choosing the one dominant field that survives.
	out := flds[:0]
	for adv, idx := 0, 0; idx < len(flds); idx += adv {
		// One iteration per name.
		// Find the sequence of fields with the name of this first field.
		fld := flds[idx]
		name := fld.name
		for adv = 1; idx+adv < len(flds); adv++ {
			fld := flds[idx+adv]
			if fld.name != name {
				break
			}
		}
		if adv == 1 { // Only one field with this name
			out = append(out, fld)
			continue
		}
		dom, ok := dominant(flds[idx : idx+adv])
		if ok {
			out = append(out, dom)
		}
	}
	flds = out
	sort.Sort(byIdx(flds))

	az95 := true
	for idx := range flds {
		fld := &flds[idx]
		az95 = az95 && isAZ95(fld.name)
		flw := followType(typ, fld.idxs)
		fld.dec, _ = decs.get(flw)
		fld.enc, _ = encs.get(flw)

		fld.nameJSON = make([]byte, 0, len(fld.name)+2)
		fld.nameJSON = append(fld.nameJSON, ',')
		fld.nameJSON = appendString(fld.nameJSON, fld.name, false)
		fld.nameJSON = append(fld.nameJSON, ':')

		fld.nameHTML = make([]byte, 0, len(fld.name)+2)
		fld.nameHTML = append(fld.nameHTML, ',')
		fld.nameHTML = appendString(fld.nameHTML, fld.name, true)
		fld.nameHTML = append(fld.nameHTML, ':')
	}

	ret := fields{
		flds:    flds,
		fldsMap: makePerf(len(flds)),
		caseMap: makePerf(len(flds)),
	}

	tups := make([]tuple, 0, len(flds))
	for idx, fld := range flds {
		tups = append(tups, tuple{
			key: []byte(fld.name),
			elm: &flds[idx],
		})
	}
	ret.fldsMap.set(tups)

	fst := make(map[string]struct{}, len(tups))
	dst := tups[:0]
	for idx := range tups {
		var buf [12]byte
		up := appendUpper(buf[:0], tups[idx].key)
		if _, ok := fst[string(up)]; !ok {
			fst[string(up)] = struct{}{}
			dst = append(dst, tuple{
				key: up,
				elm: tups[idx].elm,
			})
			continue
		}
		ret.flg |= flagDup
	}
	ret.caseMap.set(dst)

	if az95 {
		ret.flg |= flagAZ95
	}
	return ret
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

func isAZ95(str string) bool {
	const lenAZ = 'Z' - 'A'
	for idx := range str {
		if str[idx] != '_' && str[idx]&^0x20-'A' > lenAZ {
			return false
		}
	}
	return true
}

// dominant looks through the fields, all of which are known to
// have the same name, to find the single field that dominates the
// others using Go's embedding rules, modified by the presence of
// JSON tags. If there are multiple top-level fields, the boolean
// will be false: This condition is an error in Go and we skip all
// the fields.
func dominant(flds []field) (field, bool) {
	// The fields are sorted in increasing index-length order, then by presence of tag.
	// That means that the first field is the dominant one. We need only check
	// for error cases: two fields at top level, either both tagged or neither tagged.
	if len(flds) > 1 && len(flds[0].idxs) == len(flds[1].idxs) && (flds[0].flg^flds[1].flg)&flagTag == 0 {
		return field{}, false
	}
	return flds[0], true
}

func followType(typ reflect.Type, idxs []int) reflect.Type {
	for _, idx := range idxs {
		if typ.Kind() == reflect.Pointer {
			typ = typ.Elem()
		}
		typ = typ.Field(idx).Type
	}
	return typ
}

func followValue(val reflect.Value, idxs []int) (reflect.Value, error) {
	const tmpl = "cannot set embedded pointer to unexported struct: "
	for _, idx := range idxs {
		if val.Kind() == reflect.Pointer {
			if val.IsNil() {
				// If a struct embeds a pointer to an unexported type,
				// it is not possible to set a newly allocated value
				// since the field is unexported.
				//
				// See https://golang.org/issue/21357
				elm := val.Type().Elem()
				if !val.CanSet() {
					return reflect.Value{}, fieldError(tmpl + elm.String())
				}
				val.Set(reflect.New(elm))
			}
			val = val.Elem()
		}
		val = val.Field(idx)
	}
	return val, nil
}
