package multiplex

import (
	"fmt"
	"reflect"
)

type collector[T any] interface {
	Collect(T)
}

type NilHandling int

const (
	skipNilFields NilHandling = iota
	createNilFields
	panicNilFields
)

type Settings struct {
	nilHandling NilHandling
}

type Option func(*Settings)

func OptSkipNilFields(s *Settings)   { s.nilHandling = skipNilFields }
func OptCreateNilFields(s *Settings) { s.nilHandling = createNilFields }
func OptPanicNilFields(s *Settings)  { s.nilHandling = panicNilFields }

func makeSettings(opts ...Option) Settings {
	s := &Settings{
		nilHandling: createNilFields,
	}
	for _, o := range opts {
		o(s)
	}
	return *s
}

func Interface[T any, C collector[T]](maybe any, collection C, options ...Option) T {
	s := makeSettings(options...)
	tType := reflect.TypeOf(new(T)).Elem()
	out, ok := any(collection).(T)
	if !ok {
		panic("collection must implement T")
	}
	t := reflect.TypeOf(maybe)
	if t.Kind() != reflect.Pointer {
		return out
	}
	v := reflect.ValueOf(maybe).Elem()
	t = t.Elem()
	if t.Kind() != reflect.Struct {
		return out
	}

	// Collect the struct fields in the order specified in code.
field:
	for i := 0; i < t.NumField(); i++ {
		fieldVal := v.Field(i)
		if fieldVal.Kind() == reflect.Pointer {
			if fieldVal.IsNil() {
				switch s.nilHandling {
				case panicNilFields:
					panic(fmt.Sprintf("field %q is nil", t.Field(i).Name))
				case createNilFields:
					fieldVal.Set(reflect.New(fieldVal.Type().Elem()))
				case skipNilFields:
					break field
				}
			}
		} else {
			fieldVal = fieldVal.Addr()
		}
		if fieldVal.Type().AssignableTo(tType) {
			collection.Collect(fieldVal.Interface().(T))
		}
	}

	// If maybe is also a T, add it to the collection as well.
	if f, ok := maybe.(T); ok {
		collection.Collect(f)
	}

	return out
}
