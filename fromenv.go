// Copyright 2017 Alfred Landrum. All rights reserved.
// Use of this source code is governed by the license
// found in the LICENSE.txt file.

// Package fromenv can set specially tagged struct fields with values
// from the environment.
//
//	var c struct {
// 		Field1 string  	`env:"KEY1=my-default"`
// 		Field2 int     	`env:"KEY2=7"`
// 		Field3 bool    	`env:"KEY3"`
// 		Inner struct {
// 			Field4 string	`env:"KEY4"`
// 		}
// 	}
//
// 	os.Setenv("KEY1","foo")
// 	os.Unsetenv("KEY2") // show default usage
// 	os.Setenv("KEY3","true") // or 1, "1", etc.
// 	os.Setenv("KEY4","inner too!")
//
// 	err := fromenv.Unmarshal(&c)
// 	// c.Field1 == "foo"
// 	// c.Field2 == 7
// 	// c.Field3 == true
// 	// c.Inner.Field4 == "inner too!"
//
// 	// Use Map to get values from map[string]string instead:
// 	m := map[string]string{"KEY1": "bar"}
// 	err := fromenv.Unmarshal(&c, fromenv.Map(m))
// 	// c.Field1 == "bar"
// 	// c.Field2 == 7
// 	// ...
package fromenv

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
)

// Unmarshal takes a pointer to a struct, recursively looks for struct
// fields with a "env" tag, and sets the field to the value of the
// environment variable given in the tag. An env tag may optionally
// specify a default value; the field will be set to this value if the
// environment variable is not present.
//
// By default, the "os.LookupEnv" function is used to find the value
// for an environment variable. See "Map" for an example of using a
// different lookup technique.
//
// Basic types supported are: string, bool, int, uint8, uint16, uint32,
// uint64, int, int8, int16, int32, int64, float32, float64.
//
// Additionally, any type that has a `Set(string) error` method is also
// supported. This includes any type that satisfies flag.Value.
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
	return visit(in, func(c cursor) error {
		key, defval := parseTag(c.field)
		if len(key) == 0 {
			return nil
		}

		val, err := config.looker(key)
		if err != nil {
			goto reterr
		}

		if val == nil {
			if defval == nil {
				return nil
			}
			val = defval
		}

		err = setValue(c.value, *val)
		if err != nil {
			goto reterr
		}

		return nil

	reterr:
		err = fmt.Errorf("%s: field %v (%v) in struct %v", err.Error(),
			c.field.Name, c.value.Kind().String(), c.structType.Name())
		return err
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

// An Option is a functional option for Unmarshal.
type Option func(*config)

func isStructPtr(i interface{}) bool {
	r := reflect.ValueOf(i)
	if r.Kind() == reflect.Ptr && !r.IsNil() {
		return r.Elem().Kind() == reflect.Struct
	}
	return false
}

func osLookup(key string) (val *string, err error) {
	if v, ok := os.LookupEnv(key); ok {
		val = &v
	}
	return
}

type config struct {
	looker LookupEnvFunc
}

const (
	tagName = "env"
	tagSep  = "="
)

// parseTag returns the environment key and possible default value
// encoded in the field struct tag.
func parseTag(field *reflect.StructField) (string, *string) {
	tag := field.Tag.Get(tagName)
	s := strings.SplitN(tag, tagSep, 2)
	if len(s) == 1 {
		return s[0], nil
	}
	return s[0], &s[1]
}

type cursor struct {
	structType reflect.Type
	field      *reflect.StructField
	value      reflect.Value
}

// visit executes visitor on any reachable struct fields from input.
func visit(in interface{}, visitor func(cursor) error) error {
	prev := make(map[reflect.Value]bool)
	q := []reflect.Value{reflect.ValueOf(in)}

	for len(q) != 0 {
		var v reflect.Value
		v, q = q[0], q[1:]
		st, ok := settableStructPtr(v)
		if !ok || prev[st] {
			continue
		}
		prev[st] = true

		stType := st.Type()
		nfields := stType.NumField()
		for i := 0; i < nfields; i++ {
			field := stType.Field(i)
			value := st.Field(i)
			err := visitor(cursor{stType, &field, value})
			if err != nil {
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
func setValue(value reflect.Value, str string) (err error) {
	if !value.CanSet() {
		return errors.New("unsettable field")
	}

	// Support the flag package's Value interface of Set(string):
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

	return errors.New("unsupported type")
}

type setter interface {
	Set(string) error
}

func isSetter(value reflect.Value) (setter, bool) {
	i := value.Addr().Interface()
	s, ok := i.(setter)
	return s, ok
}
