// Copyright 2017 Alfred Landrum. All rights reserved.
// Use of this source code is governed by the license
// found in the LICENSE.txt file.

// Package fromenv can set specially tagged struct fields with values
// from the environment.
//
//	var c struct {
// 		Field1 string  	`fromenv:"FIELD1_KEY,my-default"`
// 		Field2 int     	`fromenv:"FIELD2_KEY,7"`
// 		Field3 bool    	`fromenv:"FIELD3_KEY"`
//
// 		Inner struct {
// 			Field4 string	`fromenv:"FIELD4_KEY"`
// 		}
// 	}
//
// 	os.Setenv("FIELD1_KEY","foo")
// 	os.Unsetenv("FIELD2_KEY")
// 	os.Setenv("FIELD3_KEY","true")
// 	os.Setenv("FIELD4_KEY","inner too!")
//
// 	err := fromenv.Unmarshal(&c)
// 	// c.Field1 == "foo"
// 	// c.Field2 == 7
// 	// c.Field3 == true
// 	// c.Inner.Field4 == "inner too!"
//
package fromenv

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
)

var (
	tagName = "fromenv"
)

// Unmarshal takes a pointer to a struct, recursively looks for struct
// fields with a "fromenv" tag, and sets the field to the value of the
// environment variable given in the tag. A fromenv tag may optionally
// specify a default value; the field will be set to this value if the
// environment variable is not present.
//
// By default, the "os.LookupEnv" function is used to find the value
// for an environment variable.
//
// Basic types supported are: string, bool, int, uint8, uint16, uint32,
// uint64, int, int8, int16, int32, int64, float32, float64.
//
// The flag package's Value interface is also supported.
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
	// processing any fields with the "fromenv" struct tag.
	return visit(in, func(c cursor) error {
		value, err := config.lookup(c.field)
		if err != nil || value == nil {
			return err
		}

		return setField(c, *value)
	})
}

// Map configures Unmarshal to use the given map for environment lookups.
func Map(m map[string]string) Option {
	return func(c *config) {
		c.looker = func(k string) (*string, error) {
			if v, ok := m[k]; ok {
				return &v, nil
			}
			return nil, nil
		}
	}
}

// DefaultsOnly configures Unmarshal to only set fields with a tag-defined
// default to that default, ignoring other fields and the environment.
func DefaultsOnly() Option {
	return Map(nil)
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

// An Option is a functional option for Unmarshal.
type Option func(*config)

func isStructPtr(i interface{}) bool {
	r := reflect.ValueOf(i)
	if r.Kind() == reflect.Ptr && !r.IsNil() {
		r = r.Elem()
		return r.Kind() == reflect.Struct
	}
	return false
}

func osLookup(key string) (*string, error) {
	if val, ok := os.LookupEnv(key); ok {
		return &val, nil
	}
	return nil, nil
}

type config struct {
	looker LookupEnvFunc
}

// lookup parses the tag, looks up the corresponding environment variable,
// and returns a pointer to its value, or a pointer to its default value
// if the variable isn't present in the environment, or nil otherwise.
func (c *config) lookup(field *reflect.StructField) (val *string, err error) {
	key, defval := parseTag(field)
	if len(key) != 0 {
		val, err = c.looker(key)
		if val == nil {
			val = defval
		}
	}
	return
}

// parseTag returns the environment key and possible default value
// encoded in the field struct tag.
func parseTag(field *reflect.StructField) (string, *string) {
	tag := field.Tag.Get(tagName)
	s := strings.SplitN(tag, ",", 2)
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

// visit executes the visitor pattern on any reachable struct fields
// starting from input.
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
func setField(c cursor, str string) error {
	if !c.value.CanSet() {
		return setErr(c, errors.New("tag found on unsettable field"))
	}

	// Support the flag package's Value interface of Set(string):
	if fv, ok := toFlagValue(c); ok {
		return setErr(c, fv.Set(str))
	}

	switch c.value.Kind() {
	case reflect.String:
		c.value.SetString(str)
		return nil

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		x, err := strconv.ParseInt(str, 0, c.value.Type().Bits())
		c.value.SetInt(x)
		return setErr(c, err)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		x, err := strconv.ParseUint(str, 0, c.value.Type().Bits())
		c.value.SetUint(x)
		return setErr(c, err)

	case reflect.Float64, reflect.Float32:
		x, err := strconv.ParseFloat(str, c.value.Type().Bits())
		c.value.SetFloat(x)
		return setErr(c, err)

	case reflect.Bool:
		x, err := strconv.ParseBool(str)
		c.value.SetBool(x)
		return setErr(c, err)
	}

	return setErr(c, errors.New("tag found on unsupported type"))
}

func toFlagValue(c cursor) (flag.Value, bool) {
	i := c.value.Addr().Interface()
	v, ok := i.(flag.Value)
	return v, ok
}

func setErr(c cursor, err error) error {
	if err != nil {
		err = fmt.Errorf("%s: field %v (%v) in struct %v",
			err.Error(), c.field.Name,
			c.value.Kind().String(), c.structType.Name())
	}
	return err
}
