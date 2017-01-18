// Copyright 2017 Alfred Landrum. All rights reserved.
// Use of this source code is governed by the license
// found in the LICENSE.txt file.

package fromenv

import (
	"errors"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestString(t *testing.T) {
	t.Parallel()

	env := map[string]string{
		"k1": "k1-val",
	}

	type S1 struct {
		Str1 string `fromenv:"k1"`
	}

	var s1 S1
	err := Unmarshal(&s1, Map(env))
	require.NoError(t, err)
	require.Equal(t, "k1-val", s1.Str1)

	type S2 struct {
		Str1 string `fromenv:"k1,not-used-default"`
	}

	var s2 S2
	err = Unmarshal(&s2, Map(env))
	require.NoError(t, err)
	require.Equal(t, "k1-val", s2.Str1)

	type S3 struct {
		Str1 string `fromenv:"nokey,def-val,with-comma"`
	}

	var s3 S3
	err = Unmarshal(&s3, Map(env))
	require.NoError(t, err)
	require.Equal(t, "def-val,with-comma", s3.Str1)
}

func TestInt(t *testing.T) {
	t.Parallel()

	env := map[string]string{
		"k1": "1",
		"k2": "i-am-not-an-int",
	}

	type S1 struct {
		Int1 int `fromenv:"k1"`
	}

	var s1 S1
	err := Unmarshal(&s1, Map(env))
	require.NoError(t, err)
	require.Equal(t, int(1), s1.Int1)

	type S2 struct {
		Int2 int `fromenv:"k2"`
	}

	var s2 S2
	err = Unmarshal(&s2, Map(env))
	require.EqualError(t, err, "strconv.ParseInt: parsing \"i-am-not-an-int\": invalid syntax: field Int2 (int) in struct S2")
}

func TestUint(t *testing.T) {
	t.Parallel()

	env := map[string]string{
		"k1": "1",
		"k2": "-1",
	}

	type S1 struct {
		Uint1 uint `fromenv:"k1"`
	}

	var s1 S1
	err := Unmarshal(&s1, Map(env))
	require.NoError(t, err)
	require.Equal(t, uint(1), s1.Uint1)

	type S2 struct {
		Uint2 uint `fromenv:"k2"`
	}

	var s2 S2
	err = Unmarshal(&s2, Map(env))
	require.EqualError(t, err, "strconv.ParseUint: parsing \"-1\": invalid syntax: field Uint2 (uint) in struct S2")
}

func TestFloat(t *testing.T) {
	t.Parallel()

	env := map[string]string{
		"k1": "1.5",
		"k2": "not-a-float",
	}

	type S1 struct {
		F1 float64 `fromenv:"k1"`
	}

	var s1 S1
	err := Unmarshal(&s1, Map(env))
	require.NoError(t, err)
	require.Equal(t, float64(1.5), s1.F1)

	type S2 struct {
		F2 float64 `fromenv:"k2"`
	}

	var s2 S2
	err = Unmarshal(&s2, Map(env))
	require.EqualError(t, err, "strconv.ParseFloat: parsing \"not-a-float\": invalid syntax: field F2 (float64) in struct S2")
}

func TestBool(t *testing.T) {
	t.Parallel()

	env := map[string]string{
		"k1": "true",
		"k2": "i-am-not-a-bool",
	}

	type S1 struct {
		Bool1 bool `fromenv:"k1"`
	}

	var s1 S1
	err := Unmarshal(&s1, Map(env))
	require.NoError(t, err)
	require.True(t, s1.Bool1)

	type S2 struct {
		Bool2 bool `fromenv:"k2"`
	}

	var s2 S2
	err = Unmarshal(&s2, Map(env))
	require.EqualError(t, err, "strconv.ParseBool: parsing \"i-am-not-a-bool\": invalid syntax: field Bool2 (bool) in struct S2")
}

type testFlagValue struct {
	x bool
}

func (tfv *testFlagValue) Set(s string) error {
	if s == "a-setter" {
		tfv.x = true
		return nil
	}
	return errors.New("not-a-setter")
}

func (tfv *testFlagValue) String() string {
	return strconv.FormatBool(tfv.x)
}

func TestSetter(t *testing.T) {
	t.Parallel()

	env := map[string]string{
		"k1": "a-setter",
		"k2": "not-a-setter",
	}
	type testSetter struct{}

	type S1 struct {
		TFV testFlagValue `fromenv:"k1"`
	}

	var s1 S1
	err := Unmarshal(&s1, Map(env))
	require.NoError(t, err)
	require.True(t, s1.TFV.x)

	type S2 struct {
		TFV testFlagValue `fromenv:"k2"`
	}

	var s2 S2
	err = Unmarshal(&s2, Map(env))
	require.EqualError(t, err, "not-a-setter: field TFV (struct) in struct S2")
}

func TestURL(t *testing.T) {
	t.Parallel()

	env := map[string]string{
		"k1": "https://docker.com/path/foo",
		"k2": "not-a-url",
	}

	type S1 struct {
		U URL `fromenv:"k1"`
	}

	var s1 S1
	err := Unmarshal(&s1, Map(env))
	require.NoError(t, err)
	require.Equal(t, env["k1"], s1.U.String())

	type S2 struct {
		U URL `fromenv:"k2"`
	}

	var s2 S2
	err = Unmarshal(&s2, Map(env))
	require.EqualError(t, err, "parse not-a-url: invalid URI for request: field U (struct) in struct S2")
}
