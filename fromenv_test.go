// Copyright 2017 Alfred Landrum. All rights reserved.
// Use of this source code is governed by the license
// found in the LICENSE.txt file.

package fromenv

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func noLookup() func(*config) {
	f := func(string) (*string, error) {
		panic("unexpected lookup in test")
	}
	return Looker(f)
}

func TestLookupConfig(t *testing.T) {
	t.Parallel()

	var err error

	type S2 struct {
		Str1 string `env:"k1=Str1-default"`
	}
	var s2 S2
	err = Unmarshal(&s2, DefaultsOnly())
	require.NoError(t, err)
	require.Equal(t, "Str1-default", s2.Str1)

	type S3 struct {
		Str1 string `env:"k1"`
	}
	var s3 S3
	badlookup := func(k string) (*string, error) {
		require.Equal(t, "k1", k)
		return nil, errors.New("lookup error")
	}
	err = Unmarshal(&s3, Looker(badlookup))
	require.EqualError(t, err, "lookup error: field Str1 (string) in struct S3")
}

func TestVisitLoop(t *testing.T) {
	t.Parallel()

	var err error
	env := map[string]string{
		"k1": "k1-val",
	}

	type S struct {
		Str1 string `env:"k1"`
		Sp   *S
	}

	// Setup loop and verify that configure halts.
	var s1, s2 S
	s1.Sp = &s2
	s2.Sp = &s1

	err = Unmarshal(&s1, Map(env))
	require.NoError(t, err)
	require.Equal(t, s1.Str1, "k1-val")
}

func TestTypeLogic(t *testing.T) {
	t.Parallel()

	var err error
	err = Unmarshal(nil, noLookup())
	require.EqualError(t, err, "passed non-pointer or nil pointer")

	type S0 struct{}
	var s0 S0
	err = Unmarshal(s0, noLookup())
	require.EqualError(t, err, "passed non-pointer or nil pointer")

	type S1 struct {
		notag int
	}
	var s1 S1
	err = Unmarshal(&s1, noLookup())
	require.NoError(t, err)

	type S2 struct {
		nonexported int `env:"k1"`
	}
	var s2 S2
	err = Unmarshal(&s2, Map(map[string]string{"k1": "k1-val"}))
	require.EqualError(t, err, "unsettable field: field nonexported (int) in struct S2")

	type S3 struct {
		Nonsupported interface{} `env:"k1"`
	}
	var s3 S3
	err = Unmarshal(&s3, Map(map[string]string{"k1": "k1-val"}))
	require.EqualError(t, err, "unsupported type: interface {}: field Nonsupported (interface) in struct S3")

	type S4 struct {
		S4Str string `env:"S4Str"`
	}
	type S5 struct {
		S5Str string `env:"S5Str"`
	}
	type S6 struct {
		S4       S4
		S5ptr    *S5
		S5nilptr *S5
	}
	env6 := map[string]string{
		"S4Str": "S4Str-val",
		"S5Str": "S5Str-val",
	}
	s6 := S6{
		S5nilptr: nil,
		S5ptr:    &S5{},
	}
	err = Unmarshal(&s6, Map(env6))
	require.NoError(t, err)
	require.Equal(t, env6["S4Str"], s6.S4.S4Str)
	require.Equal(t, env6["S5Str"], s6.S5ptr.S5Str)
	require.Nil(t, s6.S5nilptr)

	type S7 struct {
		S7str string `env:"S7str=S7default"`
	}
	type S8 struct {
		S8str string `env:"S8str=S8default"`
		S7    S7
	}
	type S9 struct {
		unexportedS8 S8
	}
	env10 := map[string]string{
		"S7str": "S7str-val",
		"S8str": "S8str-val",
	}
	var s9 S9
	err = Unmarshal(&s9, Map(env10))
	require.NoError(t, err)
	require.Empty(t, s9.unexportedS8.S7.S7str)
	require.Empty(t, s9.unexportedS8.S8str)
}

func TestRealEnvironment(t *testing.T) {
	keys := []string{
		"fromenv_test_key1",
		"fromenv_test_key2",
	}
	unsetKeys := func() {
		for _, k := range keys {
			os.Unsetenv(k)
		}
	}
	defer unsetKeys()

	var err error
	type S struct {
		Str1 string `env:"fromenv_test_key1"`
		Str2 string `env:"fromenv_test_key2=str2-def"`
	}

	unsetKeys()
	var s1 S
	err = Unmarshal(&s1)
	require.NoError(t, err)
	require.Equal(t, s1.Str1, "")
	require.Equal(t, s1.Str2, "str2-def")

	unsetKeys()
	os.Setenv("fromenv_test_key1", "key1-value")
	os.Setenv("fromenv_test_key2", "key2-value")
	var s2 S
	err = Unmarshal(&s2)
	require.NoError(t, err)
	require.Equal(t, s2.Str1, "key1-value")
	require.Equal(t, s2.Str2, "key2-value")
}

func TestString(t *testing.T) {
	t.Parallel()

	env := map[string]string{
		"k1": "k1-val",
	}

	type S1 struct {
		Str1 string `env:"k1"`
	}

	var s1 S1
	err := Unmarshal(&s1, Map(env))
	require.NoError(t, err)
	require.Equal(t, "k1-val", s1.Str1)

	type S2 struct {
		Str1 string `env:"k1=not-used-default"`
	}

	var s2 S2
	err = Unmarshal(&s2, Map(env))
	require.NoError(t, err)
	require.Equal(t, "k1-val", s2.Str1)

	type S3 struct {
		Str1 string `env:"nokey=def-val=with-sep"`
	}

	var s3 S3
	err = Unmarshal(&s3, Map(env))
	require.NoError(t, err)
	require.Equal(t, "def-val=with-sep", s3.Str1)
}

func TestInt(t *testing.T) {
	t.Parallel()

	env := map[string]string{
		"k1": "1",
		"k2": "i-am-not-an-int",
	}

	type S1 struct {
		Int1 int `env:"k1"`
	}

	var s1 S1
	err := Unmarshal(&s1, Map(env))
	require.NoError(t, err)
	require.Equal(t, int(1), s1.Int1)

	type S2 struct {
		Int2 int `env:"k2"`
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
		Uint1 uint `env:"k1"`
	}

	var s1 S1
	err := Unmarshal(&s1, Map(env))
	require.NoError(t, err)
	require.Equal(t, uint(1), s1.Uint1)

	type S2 struct {
		Uint2 uint `env:"k2"`
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
		F1 float64 `env:"k1"`
	}

	var s1 S1
	err := Unmarshal(&s1, Map(env))
	require.NoError(t, err)
	require.Equal(t, float64(1.5), s1.F1)

	type S2 struct {
		F2 float64 `env:"k2"`
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
		Bool1 bool `env:"k1"`
	}

	var s1 S1
	err := Unmarshal(&s1, Map(env))
	require.NoError(t, err)
	require.True(t, s1.Bool1)

	type S2 struct {
		Bool2 bool `env:"k2"`
	}

	var s2 S2
	err = Unmarshal(&s2, Map(env))
	require.EqualError(t, err, "strconv.ParseBool: parsing \"i-am-not-a-bool\": invalid syntax: field Bool2 (bool) in struct S2")
}

type testSetIface struct {
	x bool
}

func (tfv *testSetIface) Set(s string) error {
	if s == "a-setter" {
		tfv.x = true
		return nil
	}
	return errors.New("a-failing-setter")
}

func TestSetter(t *testing.T) {
	t.Parallel()

	env := map[string]string{
		"k1": "a-setter",
		"k2": "not-a-setter",
	}

	type S1 struct {
		TSI testSetIface `env:"k1"`
	}

	var s1 S1
	err := Unmarshal(&s1, Map(env))
	require.NoError(t, err)
	require.True(t, s1.TSI.x)

	type S2 struct {
		TSI testSetIface `env:"k2"`
	}

	var s2 S2
	err = Unmarshal(&s2, Map(env))
	require.EqualError(t, err, "a-failing-setter: field TSI (struct) in struct S2")
}

func TestSetFunc(t *testing.T) {
	t.Parallel()

	t.Run("badfuncs", func(t *testing.T) {
		t.Parallel()

		badfuncs := []interface{}{
			"hello",
			func() {},
			func(x int) {},
			func(x int) error { return nil },
			func(x int, y string) error { return nil },
			func(x *int, y int) error { return nil },
			func(x *int, y string) int { return 0 },
		}
		b0 := struct{}{}
		for i := range badfuncs {
			require.Panics(t, func() { Unmarshal(&b0, SetFunc(badfuncs[i])) })
		}
	})

	t.Run("simple", func(t *testing.T) {
		t.Parallel()

		durSetter := func(d *time.Duration, s string) error {
			x, err := time.ParseDuration(s)
			*d = x
			return err
		}

		env := map[string]string{
			"k1": "5s",
			"k2": "not-a-duration",
		}

		type S1 struct {
			D time.Duration `env:"k1"`
		}

		var s1 S1
		err := Unmarshal(&s1, Map(env), SetFunc(durSetter))
		require.NoError(t, err)
		require.Equal(t, s1.D, 5*time.Second)

		type S2 struct {
			D time.Duration `env:"k2"`
		}

		var s2 S2
		err = Unmarshal(&s2, Map(env), SetFunc(durSetter))
		require.EqualError(t, err, "time: invalid duration not-a-duration: field D (int64) in struct S2")
	})

	t.Run("pointer", func(t *testing.T) {
		t.Parallel()

		durSetter := func(d *time.Duration, s string) error {
			x, err := time.ParseDuration(s)
			*d = x
			return err
		}

		env := map[string]string{
			"k1": "5s",
		}

		type S1 struct {
			D *time.Duration `env:"k1"`
		}

		var s1 S1
		err := Unmarshal(&s1, Map(env), SetFunc(durSetter))
		require.NoError(t, err)
		require.Equal(t, *s1.D, 5*time.Second)
	})

}
