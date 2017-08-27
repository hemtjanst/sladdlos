package tradfri

import (
	"encoding/json"
	//"log"
	"reflect"
	"strconv"
)

type ObservableCallback func(change []*ObservedChange)
type ObserveFilter func(observation *ObservedChange) bool

type observable struct {
	observers []ObservableCallback
}

func (o *observable) Observe(callback ObservableCallback) {
	if o.observers == nil {
		o.observers = []ObservableCallback{}
	}
	o.observers = append(o.observers, callback)
}

func (o *observable) ObserveFilter(filter []ObserveFilter, callback ObservableCallback) {
	o.observers = append(o.observers, func(change []*ObservedChange) {
		ch := []*ObservedChange{}
		for _, v := range change {
			r := true
			for _, f := range filter {
				if !f(v) {
					r = false
					break
				}
			}
			if r {
				ch = append(ch, v)
			}
		}
		if len(ch) > 0 {
			callback(ch)
		}
	})
}

func (o *observable) OnChange(change []*ObservedChange) {
	if o == nil {
		//log.Print("Called OnChange() on nil!")
		return
	}
	if o.observers == nil {
		//log.Printf("Called OnChange() with nil observers")
		return
	}
	for _, b := range o.observers {
		b(change)
	}
}

type ObservedChange struct {
	Path     string
	Field    string
	OldValue interface{}
	NewValue interface{}
}

func deepFields(iface interface{}) []string {
	ret := make([]string, 0)
	ifv := reflect.ValueOf(iface)
	ift := reflect.TypeOf(iface)

	for i := 0; i < ift.NumField(); i++ {
		v := ifv.Field(i)
		n := ifv.Type().Field(i).Name
		ign := ifv.Type().Field(i).Tag.Get("json")
		if ign == "-" {
			continue
		}

		if byte(n[0])&0x60 != 0x40 {
			// Field is not exported
			continue
		}

		switch v.Kind() {
		case reflect.Struct:
			ret = append(ret, deepFields(v.Interface())...)
		default:
			ret = append(ret, n)
		}
	}
	return ret
}

func comparePtr(fOld, fNew reflect.Value, prefix, fName string) (updates []*ObservedChange) {
	updates = []*ObservedChange{}
	if fOld.IsNil() && fNew.IsNil() {
		return
	}
	if fNew.IsNil() {
		updates = append(updates, &ObservedChange{
			Path:     prefix,
			Field:    fName,
			OldValue: fOld.Interface(),
			NewValue: nil,
		})
		fOld.Set(fNew)
		return
	}
	if fOld.IsNil() {
		fOld.Set(reflect.New(fNew.Elem().Type()))
	}
	if prefix != "" {
		prefix = prefix + "/"
	}
	updates = append(updates, compare(fOld, fNew, prefix+fName)...)
	return
}

func compare(oldValue, newValue reflect.Value, prefix string) (updates []*ObservedChange) {
	updates = []*ObservedChange{}
	oldValue = reflect.Indirect(oldValue)
	newValue = reflect.Indirect(newValue)
	if !oldValue.IsValid() {
		return
	}
	//log.Printf("Parsing: %v", oldValue.Interface())
	fields := deepFields(oldValue.Interface())
	for _, fName := range fields {
		if byte(fName[0])&0x60 != 0x40 {
			// Field is not exported
			continue
		}
		fOld := oldValue.FieldByName(fName)
		fNew := newValue.FieldByName(fName)

		switch fOld.Type().Kind() {
		case reflect.Ptr:
			if fOld.Type().Elem().Kind() == reflect.Struct {
				updates = append(updates, comparePtr(fOld, fNew, prefix, fName)...)
				continue
			}
		case reflect.Slice:
			if fOld.Type().Elem().Kind() == reflect.Ptr {
				if fOld.IsNil() && fNew.IsNil() {
					continue
				}
				if fNew.IsNil() {
					updates = append(updates, &ObservedChange{
						Path:     prefix,
						Field:    fName,
						OldValue: fOld.Interface(),
						NewValue: nil,
					})
					fOld.Set(fNew)
					continue
				}
				if fOld.IsNil() {
					updates = append(updates, &ObservedChange{
						Path:     prefix,
						Field:    fName,
						OldValue: nil,
						NewValue: fNew.Interface(),
					})
					fOld.Set(fNew)
					continue
				}
				if fOld.Len() != fNew.Len() {
					updates = append(updates, &ObservedChange{
						Path:     prefix,
						Field:    fName,
						OldValue: fOld.Interface(),
						NewValue: fNew.Interface(),
					})
					fOld.Set(fNew)
					continue
				}

				sPrefix := prefix
				if sPrefix != "" {
					sPrefix = sPrefix + "/"
				}
				for i := 0; i < fOld.Len(); i++ {
					updates = append(updates, comparePtr(fOld.Index(i), fNew.Index(i), sPrefix+fName, strconv.Itoa(i))...)
				}

				continue
			}
		}
		if !reflect.DeepEqual(fOld.Interface(), fNew.Interface()) {
			updates = append(updates, &ObservedChange{
				Path:     prefix,
				Field:    fName,
				OldValue: fOld.Interface(),
				NewValue: fNew.Interface(),
			})
			fOld.Set(fNew)
		}
	}
	return
}

func update(j []byte, o interface{}) error {
	oldValue := reflect.ValueOf(o)

	m := oldValue.MethodByName("OnChange")
	/*if !m.IsValid() {
		return json.Unmarshal(j, oldValue.Interface())
	}*/

	newValue := reflect.New(reflect.TypeOf(o).Elem())
	err := json.Unmarshal(j, newValue.Interface())
	if err != nil {
		return err
	}

	updates := compare(oldValue, newValue, "")

	m.Call([]reflect.Value{reflect.ValueOf(updates)})

	return nil
}
