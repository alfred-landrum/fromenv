// Copyright 2017 Alfred Landrum. All rights reserved.
// Use of this source code is governed by the license
// found in the LICENSE.txt file.

package fromenv

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
)

type unmarshalError struct {
	err    error
	cursor *cursor
}

func (e *unmarshalError) Error() string {
	return fmt.Sprintf("%s: field %v (%v) in struct %v", e.err.Error(),
		e.cursor.field.Name, e.cursor.value.Kind().String(), e.cursor.structType.Name())
}

// Unmarshal takes a pointer to a struct, recursively looks for struct fields
// with a "env" tag, and, by default, uses the os.LookupEnv function to
// determine the desired value from the environment.
//
// An env tag may optionally specify a default desired value; if no entry exists
// in the environment for the field's key, then the desired value of the field
// will be this default value.
//
// Unmarshal will set the struct field (of type T) to the desired value by whichever method matches first:
//
// * Using a function of type "func(*T, string) error" configured via SetFunc.
//
// * If T satisfies an interface of `func Set(string) error`, then its Set function.
//
// * If T is a boolean, numeric, or string type, then the appropriate strconv function will be used.
//
// Unmarshal will return an error if the env tag is used on a struct field that
// can't be set with any of the above, or if the value's setting function fails.
func Unmarshal(in interface{}, options ...Option) error {
	// The input interface should be a non-nil pointer to struct.
	if !isStructPtr(in) {
		return errors.New("passed non-pointer or nil pointer")
	}
	config := &config{
		looker: osLookup,
	}
	for _, option := range options {
		option(config)
	}

	// Visit each struct field reachable from the input interface,
	// processing any fields with the "env" struct tag.
	return visit(in, func(c *cursor) error {
		key, defval := parseTag(c)
		if len(key) == 0 {
			return nil
		}

		val, err := config.looker(key)
		if err != nil {
			return &unmarshalError{err, c}
		}

		if val == nil {
			if defval == nil {
				return nil
			}
			val = defval
		}

		err = setValue(config, c.value, *val)
		if err != nil {
			return &unmarshalError{err, c}
		}

		return nil
	})
}

// A LookupEnvFunc retrieves the value of the environment variable
// named by the key. If the variable isn't present, a nil pointer
// is returned.
type LookupEnvFunc func(key string) (value *string, err error)

// Looker configures the environment lookup function used during an
// Unmarshal call.
func Looker(f LookupEnvFunc) Option {
	return func(c *config) {
		c.looker = f
	}
}

// Map configures Unmarshal to use the given map for environment lookups.
func Map(m map[string]string) Option {
	return Looker(func(k string) (*string, error) {
		if v, ok := m[k]; ok {
			return &v, nil
		}
		return nil, nil
	})
}

// DefaultsOnly configures Unmarshal to only set fields with a tag-defined
// default to that default, ignoring other fields and the environment.
func DefaultsOnly() Option {
	return Map(nil)
}

type setFunc func(val reflect.Value, s string) error

// validateSetFunc returns ok if fn is a "func(*T, string) error", returning
// reflect.Type T and the equivalent of fn that takes a reflect.Value of type T.
func validateSetFunc(fn interface{}) (argType reflect.Type, setFn setFunc, ok bool) {
	fnValue := reflect.ValueOf(fn)
	if fnValue.Kind() != reflect.Func {
		return
	}
	fnType := fnValue.Type()
	if !(fnType.NumIn() == 2 && !fnType.IsVariadic() && fnType.NumOut() == 1) {
		return
	}
	a0 := fnType.In(0)
	if a0.Kind() != reflect.Ptr {
		return
	}

	if fnType.In(1) != reflect.TypeOf((*string)(nil)).Elem() {
		return
	}

	errIface := reflect.TypeOf((*error)(nil)).Elem()
	if !fnType.Out(0).Implements(errIface) {
		return
	}

	argType = a0.Elem()
	setFn = func(val reflect.Value, s string) error {
		rets := fnValue.Call([]reflect.Value{val.Addr(), reflect.ValueOf(s)})
		if rets[0].IsNil() {
			return nil
		}
		return rets[0].Interface().(error)
	}

	return argType, setFn, true
}

// SetFunc takes a function of form "func(*T, string) error", and configures
// Unmarshal to use that function to set the value of any type T's.
func SetFunc(fn interface{}) Option {
	return func(c *config) {
		argType, setFn, ok := validateSetFunc(fn)
		if !ok {
			panic("expected a function matching: func(*T, string) error")
		}

		if c.setFuncs == nil {
			c.setFuncs = make(map[reflect.Type]setFunc)
		}
		c.setFuncs[argType] = setFn
	}
}

// An Option is a functional option for Unmarshal.
type Option func(*config)

func isStructPtr(i interface{}) bool {
	r := reflect.ValueOf(i)
	if r.Kind() == reflect.Ptr && !r.IsNil() {
		return r.Elem().Kind() == reflect.Struct
	}
	return false
}

func osLookup(key string) (*string, error) {
	if v, ok := os.LookupEnv(key); ok {
		return &v, nil
	}
	return nil, nil
}

type config struct {
	looker   LookupEnvFunc
	setFuncs map[reflect.Type]setFunc
}

const (
	tagName = "env"
	tagSep  = "="
)

// parseTag returns the environment key and possible default value
// encoded in the field struct tag.
func parseTag(c *cursor) (string, *string) {
	tag := c.field.Tag.Get(tagName)
	s := strings.SplitN(tag, tagSep, 2)
	if len(s) == 1 {
		return s[0], nil
	}
	return s[0], &s[1]
}

type cursor struct {
	structType reflect.Type
	field      reflect.StructField
	value      reflect.Value
}

// visit executes visitor on all reachable fields from its input struct.
func visit(in interface{}, visitor func(*cursor) error) error {
	prev := make(map[reflect.Value]struct{})
	for q := []reflect.Value{reflect.ValueOf(in)} ; len(q) != 0 ; q = q[1:] {
		structPtr, ok := settableStructPtr(q[0])
		if !ok {
			continue
		}
		if _, inPrev := prev[structPtr]; inPrev {
			continue
		}
		prev[structPtr] = struct{}{}

		structType := structPtr.Type()
		n := structType.NumField()
		for i := 0; i < n; i++ {
			field := structType.Field(i)
			value := structPtr.Field(i)
			c := cursor{structType, field, value}
			if err := visitor(&c); err != nil {
				return err
			}
			q = append(q, value)
		}
	}

	return nil
}

func settableStructPtr(v reflect.Value) (reflect.Value, bool) {
	if v.Kind() == reflect.Ptr && !v.IsNil() {
		v = v.Elem()
	}
	if v.Kind() == reflect.Struct {
		return v, v.CanSet()
	}
	return reflect.Value{}, false
}

// Set the struct field at the cursor to the given string.
func setValue(cfg *config, value reflect.Value, str string) error {
	if value.Kind() == reflect.Ptr {
		if value.IsNil() {
			value.Set(reflect.New(value.Type().Elem()))
		}
		value = value.Elem()
	}

	if !value.CanSet() {
		return errors.New("unsettable field")
	}

	if setfn, ok := cfg.setFuncs[value.Type()]; ok {
		return setfn(value, str)
	}

	if s, ok := isSetter(value); ok {
		return s.Set(str)
	}

	switch value.Kind() {
	case reflect.String:
		value.SetString(str)
		return nil

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		x, err := strconv.ParseInt(str, 0, value.Type().Bits())
		value.SetInt(x)
		return err

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		x, err := strconv.ParseUint(str, 0, value.Type().Bits())
		value.SetUint(x)
		return err

	case reflect.Float64, reflect.Float32:
		x, err := strconv.ParseFloat(str, value.Type().Bits())
		value.SetFloat(x)
		return err

	case reflect.Bool:
		x, err := strconv.ParseBool(str)
		value.SetBool(x)
		return err
	}

	return fmt.Errorf("unsupported type: %v", value.Type().String())
}

type setter interface {
	Set(string) error
}

func isSetter(value reflect.Value) (setter, bool) {
	i := value.Addr().Interface()
	s, ok := i.(setter)
	return s, ok
}
