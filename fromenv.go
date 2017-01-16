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

var (
	tagName = "fromenv"
)

// Configure takes a pointer to a struct, recursively looking
// for any struct fields with the "fromenv" tag set, and sets
// the field to the value of the specified environment variable.
func Configure(in interface{}, options ...func(*config) error) error {
	// The input interface should be a non-nil pointer to struct.
	if !isStructPtr(in) {
		return errors.New("passed non-pointer or nil pointer")
	}
	config := &config{
		looker: osLookup,
	}
	for _, option := range options {
		err := option(config)
		if err != nil {
			return err
		}
	}

	// Visit each struct field reachable from the input interface,
	// processing any fields with the "fromenv" struct tag.
	return visit(in, func(structType reflect.Type, field *reflect.StructField, fieldValue reflect.Value) error {
		key, defval := parseTag(field)
		if len(key) == 0 {
			return nil
		}

		if !fieldValue.CanSet() {
			// This is likely an unexported field; see Value.CanSet().
			return fmt.Errorf("tag found on unsettable field: field %v (%v) in struct %v",
				field.Name, fieldValue.Kind().String(), structType.Name())
		}

		setter := setterFor(field.Type.Kind())
		if setter == nil {
			return fmt.Errorf("tag found on unsupported type: field %v (%v) in struct %v",
				field.Name, fieldValue.Kind().String(), structType.Name())
		}

		// Set the field's value to that retrieved from the environment.
		// If no environment value is set, and no default is specified
		// by the tag, leave the field untouched.
		val, err := config.looker(key)
		if err != nil {
			return err
		}
		if val == nil {
			if defval == nil {
				return nil
			}
			val = defval
		}
		if err := setter(fieldValue, *val); err != nil {
			return fmt.Errorf("failed to configure from %s: %s", key, err.Error())
		}
		return nil
	})
}

// A LookupEnvFunc retrieves the value of environment variable "key".
// If the key isn't found, a LookupEnvFunc should return nil,nil.
type LookupEnvFunc func(key string) (value *string, err error)

// LookupEnv sets the environment lookup function.
func LookupEnv(f LookupEnvFunc) func(*config) error {
	return func(c *config) error {
		c.looker = f
		return nil
	}
}

// LookupMap sets a map[string]string to use for environment lookups.
func LookupMap(m map[string]string) func(*config) error {
	return func(c *config) error {
		c.looker = func(k string) (*string, error) {
			v, ok := m[k]
			if ok {
				return &v, nil
			}
			return nil, nil
		}
		return nil
	}
}

// DefaultsOnly causes no environment lookups to be made, so only
// fields that specify a default value will be set.
func DefaultsOnly() func(*config) error {
	return func(c *config) error {
		c.looker = func(string) (*string, error) {
			return nil, nil
		}
		return nil
	}
}

func osLookup(key string) (*string, error) {
	val, ok := os.LookupEnv(key)
	if ok {
		return &val, nil
	}
	return nil, nil
}

type config struct {
	looker LookupEnvFunc
}

func isStructPtr(i interface{}) bool {
	r := reflect.ValueOf(i)
	if r.Kind() == reflect.Ptr && !r.IsNil() {
		r = r.Elem()
		return r.Kind() == reflect.Struct
	}
	return false
}

func structValue(v reflect.Value) (reflect.Value, bool) {
	if v.Kind() == reflect.Ptr && !v.IsNil() {
		v = v.Elem()
	}
	if v.Kind() == reflect.Struct {
		return v, true
	}
	return reflect.Value{}, false
}

// A visitFunc is called from visit(...) for each struct field.
type visitFunc func(structType reflect.Type, structField *reflect.StructField, fieldValue reflect.Value) error

// visit executes the visitor pattern on any recursively reachable
// struct fields starting from input.
func visit(in interface{}, visitFn visitFunc) error {
	prev := make(map[reflect.Value]bool)
	q := []reflect.Value{reflect.ValueOf(in)}

	for len(q) != 0 {
		var v reflect.Value
		v, q = q[0], q[1:]
		st, ok := structValue(v)
		if !ok || prev[st] {
			continue
		}
		prev[st] = true

		stType := st.Type()
		nfields := stType.NumField()
		for i := 0; i < nfields; i++ {
			field := stType.Field(i)
			value := st.Field(i)
			if err := visitFn(stType, &field, value); err != nil {
				return err
			}
			q = append(q, value)
		}
	}

	return nil
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

type fieldSetter func(field reflect.Value, s string) error

func setterFor(kind reflect.Kind) fieldSetter {
	switch kind {
	case reflect.String:
		return stringSetter
	case reflect.Int:
		return intSetter
	case reflect.Bool:
		return boolSetter
	}
	return nil
}

func stringSetter(field reflect.Value, s string) error {
	field.Set(reflect.ValueOf(s))
	return nil
}

func intSetter(field reflect.Value, s string) error {
	i, err := strconv.ParseInt(s, 0, 64)
	field.Set(reflect.ValueOf(int(i)))
	return err
}

func boolSetter(field reflect.Value, s string) error {
	b, err := strconv.ParseBool(s)
	field.Set(reflect.ValueOf(b))
	return err
}
