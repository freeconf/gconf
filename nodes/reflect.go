package nodes

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"unicode"

	"github.com/freeconf/gconf/meta"
	"github.com/freeconf/gconf/node"
	"github.com/freeconf/gconf/val"
)

// Uses reflection to marshal data into go structs.  If you want to override
// then use:
//     &nodes.Extend{
//         Base: nodes.ReflectChild(obj),
//         OnChild:...
//     }
type Reflect struct {
	OnChild OnReflectChild
	OnList  OnReflectList

	// When writing to map[], serialize enums and enum lists as one of 3 different options
	EnumFieldsAs WriteEnumAs
}

type WriteEnumAs int

const (
	WriteEnumAsEnum = iota
	WriteEnumAsId
	WriteEnumAsLabel
)

func ReflectChild(obj interface{}) node.Node {
	return Reflect{}.child(reflect.ValueOf(obj))
}

func ReflectList(obj interface{}) node.Node {
	return Reflect{}.list(reflect.ValueOf(obj), nil)
}

func (self Reflect) isEmpty(v reflect.Value) bool {
	if !v.IsValid() {
		return true
	}
	switch v.Type().Kind() {
	case reflect.Struct:
		return false
	default:
		return v.IsNil()
	}
	return false
}

type OnReflectChild func(Reflect, reflect.Value) node.Node

func (self Reflect) Child(v reflect.Value) node.Node {
	switch v.Kind() {
	case reflect.Map:
		return self.childMap(v)
	case reflect.Slice:
	case reflect.Array:
	case reflect.Interface:
		switch v.Elem().Type().Kind() {
		case reflect.Map:
			return self.childMap(v.Elem())
		}
		return self.strukt(v.Elem())
	case reflect.Ptr:
		return self.strukt(v)
	case reflect.Struct:
		return self.strukt(v.Addr())
	}
	panic("unsupported type for child container " + v.String())
}

func (self Reflect) child(v reflect.Value) node.Node {
	if self.isEmpty(v) {
		return nil
	}
	if self.OnChild != nil {
		return self.OnChild(self, v)
	}
	return self.Child(v)
}

type OnReflectList func(Reflect, reflect.Value) node.Node

func (self Reflect) List(o interface{}) node.Node {
	return self.ReflectList(reflect.ValueOf(o), nil)
}

func (self Reflect) ReflectList(v reflect.Value, onUpdate OnListValueChange) node.Node {
	switch v.Kind() {
	case reflect.Map:
		return self.listMap(v)
	case reflect.Interface:
		switch v.Elem().Kind() {
		case reflect.Slice:
			return self.listSlice(v.Elem(), onUpdate)
		case reflect.Map:
			return self.listMap(v.Elem())
		}
	case reflect.Slice:
		return self.listSlice(v, onUpdate)
	}
	panic("unsupported type for listing " + v.String())
}

func (self Reflect) list(v reflect.Value, onUpdate OnListValueChange) node.Node {
	if self.isEmpty(v) {
		return nil
	}
	if self.OnList != nil {
		return self.OnList(self, v)
	}
	return self.ReflectList(v, onUpdate)
}

type sliceEntry struct {
	key []val.Value
	n   node.Node
	pos int
}

type sliceSorter []sliceEntry

func (self sliceSorter) Less(a, b int) bool {
	return val.CompareVals(self[a].key, self[b].key) < 0
}

func (self sliceSorter) Swap(i, j int) {
	self[i], self[j] = self[j], self[i]
}

func (self sliceSorter) Len() int {
	return len(self)
}

func (self sliceSorter) findFunc(key []val.Value) func(int) bool {
	return func(i int) bool {
		return val.CompareVals(self[i].key, key) >= 0
	}
}

func (self sliceSorter) find(key []val.Value) (node.Node, int) {
	found := sort.Search(len(self), self.findFunc(key))
	if found >= 0 && found < len(self) {
		// Search find first match equal or greater, so we need to check for equality
		if val.EqualVals(self[found].key, key) {
			return self[found].n, self[found].pos
		}
	}
	return nil, -1
}

func (self Reflect) buildKeys(s node.Selection, keyMeta []meta.HasType, slce reflect.Value) (sliceSorter, error) {
	var err error
	entries := make(sliceSorter, slce.Len())
	for i := range entries {
		entries[i].pos = i
		entries[i].n = self.child(slce.Index(i))
		entries[i].key = make([]val.Value, len(keyMeta))
		for j, k := range keyMeta {
			r := node.FieldRequest{
				Meta: k.(meta.HasType),
			}
			var hnd node.ValueHandle
			if err = entries[i].n.Field(r, &hnd); err != nil {
				return nil, err
			}
			entries[i].key[j] = hnd.Val
		}
	}
	sort.Sort(entries)
	return entries, nil
}

func (self Reflect) listSlice(v reflect.Value, onChange OnListValueChange) node.Node {
	var entries sliceSorter
	e := v.Type().Elem()
	return &Basic{
		OnNext: func(r node.ListRequest) (node.Node, []val.Value, error) {
			if entries == nil {
				var err error
				entries, err = self.buildKeys(r.Selection, r.Meta.KeyMeta(), v)
				if err != nil {
					return nil, nil, err
				}
			}
			key := r.Key
			if r.New {
				item := self.create(e)
				v = reflect.Append(v, item)
				if onChange != nil {
					onChange(v)
				}
				entries = nil
				return self.child(item), key, nil
			} else if key != nil {
				if found, i := entries.find(key); found != nil {
					if r.Delete {
						part1 := v.Slice(0, i)
						part2 := v.Slice(i+1, v.Len())
						v = reflect.AppendSlice(part1, part2)
						onChange(v)
						entries = nil
						return nil, nil, nil
					}
					return found, key, nil
				}
			} else {
				if r.Row < v.Len() {
					e := entries[r.Row]
					return e.n, e.key, nil
				}
			}
			return nil, nil, nil
		},
	}
}

type valSorter []reflect.Value

func (self valSorter) Len() int {
	return len(self)
}

func (self valSorter) Less(i, j int) bool {
	switch self[i].Type().Kind() {
	case reflect.String:
		return strings.Compare(self[i].String(), self[j].String()) < 0
	case reflect.Int:
		return self[i].Int() < self[j].Int()
	}
	panic("not supported")
}

func (self valSorter) Swap(i, j int) {
	self[i], self[j] = self[j], self[i]
}

func (self Reflect) listMap(v reflect.Value) node.Node {
	var keys []reflect.Value
	e := v.Type().Elem()
	return &Basic{
		OnNext: func(r node.ListRequest) (node.Node, []val.Value, error) {
			var item reflect.Value
			key := r.Key
			if r.New {
				item = self.create(e)
				keyVal := reflect.ValueOf(key[0].Value())
				v.SetMapIndex(keyVal, item)
			} else if key != nil {
				keyVal := reflect.ValueOf(key[0].Value())
				if r.Delete {
					v.SetMapIndex(keyVal, reflect.ValueOf(nil))
					return nil, nil, nil
				}
				item = v.MapIndex(keyVal)
			} else {
				if keys == nil {
					keys = v.MapKeys()
					sort.Sort(valSorter(keys))
				}
				if r.Row < len(keys) {
					// assumes only 1 key
					keyVal := keys[r.Row]
					item = v.MapIndex(keyVal)
					var err error
					key, err = node.NewValues(r.Meta.KeyMeta(), keyVal.Interface())
					if err != nil {
						return nil, nil, err
					}
				}
			}
			if item.IsValid() {
				return self.child(item), key, nil
			}
			return nil, nil, nil
		},
	}
}

type OnListValueChange func(update reflect.Value)

func (self Reflect) childMap(v reflect.Value) node.Node {
	e := v.Type().Elem()
	return &Basic{
		Peekable: v.Interface(),
		OnChoose: func(state node.Selection, choice *meta.Choice) (m *meta.ChoiceCase, err error) {
			for _, c := range choice.Cases() {
				i := meta.Children(c)
				for i.HasNext() {
					d := i.Next().(meta.Identifiable)
					mapKey := reflect.ValueOf(d.Ident())
					mapVal := v.MapIndex(mapKey)
					if mapVal.IsValid() {
						return c, nil
					}
				}
			}
			return nil, nil
		},
		OnChild: func(r node.ChildRequest) (node.Node, error) {
			mapKey := reflect.ValueOf(r.Meta.Ident())
			var childInstance reflect.Value
			if r.New {
				childInstance = self.create(e)
				v.SetMapIndex(mapKey, childInstance)
			} else if r.Delete {
				// how you call delete(key) on map thru reflection
				v.SetMapIndex(mapKey, reflect.Value{})
				return nil, nil
			} else {
				childInstance = v.MapIndex(mapKey)
			}
			if meta.IsList(r.Meta) {
				onUpdate := func(update reflect.Value) {
					v.SetMapIndex(mapKey, update)
				}
				return self.list(childInstance, onUpdate), nil
			}
			return self.child(childInstance), nil
		},
		OnField: func(r node.FieldRequest, hnd *node.ValueHandle) error {
			mapKey := reflect.ValueOf(r.Meta.Ident())
			if r.Write {
				if r.Clear {
					v.SetMapIndex(mapKey, reflect.Value{})
				} else {
					var x interface{}
					switch r.Meta.Type().Format() {
					case val.FmtEnum:
						switch self.EnumFieldsAs {
						case WriteEnumAsId:
							x = hnd.Val.(val.Enum).Id
						case WriteEnumAsLabel:
							x = hnd.Val.(val.Enum).Label
						default:
							x = hnd.Val
						}
					case val.FmtEmptyList:
						switch self.EnumFieldsAs {
						case WriteEnumAsId:
							x = hnd.Val.(val.EnumList).Ids()
						case WriteEnumAsLabel:
							x = hnd.Val.(val.EnumList).Labels()
						default:
							x = hnd.Val
						}
					default:
						x = hnd.Val.Value()
					}
					v.SetMapIndex(mapKey, reflect.ValueOf(x))
				}
			} else {
				fval := v.MapIndex(mapKey)
				if fval.IsValid() {
					var err error
					hnd.Val, err = node.NewValue(r.Meta.Type(), fval.Interface())
					if err != nil {
						return err
					}
				}
			}
			return nil
		},
	}
}

func (self Reflect) create(t reflect.Type) reflect.Value {
	switch t.Kind() {
	case reflect.Ptr:
		return reflect.New(t.Elem())
	case reflect.Interface:
		return reflect.ValueOf(make(map[string]interface{}))
	case reflect.Map:
		return reflect.MakeMap(t)
	case reflect.Slice:
		return reflect.MakeSlice(t, 0, 0)
	}
	panic(fmt.Sprintf("creating type not supported %v", t))
}

func (self Reflect) strukt(ptrVal reflect.Value) node.Node {
	elemVal := ptrVal.Elem()
	return &Basic{
		Peekable: ptrVal.Interface(),
		OnChild: func(r node.ChildRequest) (node.Node, error) {
			fieldName := MetaNameToFieldName(r.Meta.Ident())
			childVal := elemVal.FieldByName(fieldName)
			if r.New {
				childInstance := self.create(childVal.Type())
				childVal.Set(childInstance)
			}
			if meta.IsList(r.Meta) {
				onUpdate := func(update reflect.Value) {
					childVal.Set(update)
				}
				return self.list(childVal, onUpdate), nil
			}
			return self.child(childVal), nil
		},
		OnField: func(r node.FieldRequest, hnd *node.ValueHandle) (err error) {
			if r.Write {
				err = WriteField(r.Meta, ptrVal, hnd.Val)
			} else {
				hnd.Val, err = ReadField(r.Meta, ptrVal)
			}
			return
		},
	}
}

/////////////////
func WriteField(m meta.HasType, ptrVal reflect.Value, v val.Value) error {
	return WriteFieldWithFieldName(MetaNameToFieldName(m.Ident()), m, ptrVal, v)
}

// Look for public fields that match fieldName.  Some attempt will be made to convert value to proper
// type.
//
// TODO: We only look for fields, but it would be useful to look for methods as well with pattern
// Set___(x) or the like
func WriteFieldWithFieldName(fieldName string, m meta.HasType, ptrVal reflect.Value, v val.Value) error {
	elemVal := ptrVal.Elem()
	if !elemVal.IsValid() {
		panic(fmt.Sprintf("Cannot find property \"%s\" on invalid or nil %s", fieldName, ptrVal))
	}

	fieldVal := elemVal.FieldByName(fieldName)
	if !fieldVal.IsValid() {
		panic(fmt.Sprintf("Invalid property \"%s\" on %s", fieldName, elemVal.Type()))
	}
	if v == nil {
		panic(fmt.Sprintf("No value given to set %s", m.Ident()))
	}
	switch v.Format() {
	case val.FmtEnum:
		e := v.(val.Enum)
		switch fieldVal.Kind() {
		case reflect.Int:
			fieldVal.SetInt(int64(e.Id))
		case reflect.String:
			fieldVal.SetString(e.Label)
		}
	case val.FmtEnumList:
		el := v.(val.EnumList)
		switch fieldVal.Elem().Kind() {
		case reflect.Int:
			fieldVal.Set(reflect.ValueOf(el.Ids()))
		case reflect.String:
			fieldVal.Set(reflect.ValueOf(el.Labels()))
		}
	default:
		fieldVal.Set(reflect.ValueOf(v.Value()))
	}
	return nil
}

func ReadField(m meta.HasType, ptrVal reflect.Value) (val.Value, error) {
	return ReadFieldWithFieldName(MetaNameToFieldName(m.Ident()), m, ptrVal)
}

func ReadFieldWithFieldName(fieldName string, m meta.HasType, ptrVal reflect.Value) (v val.Value, err error) {
	elemVal := ptrVal.Elem()
	if elemVal.Kind() == reflect.Ptr {
		panic(fmt.Sprintf("Pointer to a pointer not legal %s on %v ", m.Ident(), ptrVal))
	}
	fieldVal := elemVal.FieldByName(fieldName)

	if !fieldVal.IsValid() {
		panic(fmt.Sprintf("Field not found: %s on %v", m.Ident(), ptrVal))
	}

	// convert arrays to slices so casts work. this should not make a copy
	// of the array and therefore be efficient operation
	dt := m.Type()
	// Turn arrays into slices to leverage more of val.Conv's ability to convert data
	if dt.Format().IsList() && fieldVal.Kind() == reflect.Array {
		fieldVal = fieldVal.Slice(0, fieldVal.Len())
	}

	switch dt.Format() {
	case val.FmtString:
		s := fieldVal.String()
		if len(s) == 0 {
			return nil, nil
		}
		return val.String(s), nil
	case val.FmtAny:
		if fieldVal.IsNil() {
			return nil, nil
		}
	}
	return node.NewValue(dt, fieldVal.Interface())
}

func MetaNameToFieldName(in string) string {
	// assumes fix is always shorter because char can be dropped and not added
	fixed := make([]rune, len(in))
	cap := true
	j := 0
	for _, r := range in {
		if r == '-' || r == '_' {
			cap = true
		} else {
			if cap {
				fixed[j] = unicode.ToUpper(r)
			} else {
				fixed[j] = r
			}
			j += 1
			cap = false
		}
	}
	return string(fixed[:j])
}
