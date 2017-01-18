// Copyright 2017 Alfred Landrum. All rights reserved.
// Use of this source code is governed by the license
// found in the LICENSE.txt file.

package fromenv

import (
	"errors"
	"os"
	"testing"

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
		Str1 string `fromenv:"k1,Str1-default"`
	}
	var s2 S2
	err = Unmarshal(&s2, DefaultsOnly())
	require.NoError(t, err)
	require.Equal(t, "Str1-default", s2.Str1)

	type S3 struct {
		Str1 string `fromenv:"k1"`
	}
	var s3 S3
	badlookup := func(k string) (*string, error) {
		require.Equal(t, "k1", k)
		return nil, errors.New("lookup error")
	}
	err = Unmarshal(&s3, Looker(badlookup))
	require.EqualError(t, err, "lookup error")
}

func TestVisitLoop(t *testing.T) {
	t.Parallel()

	var err error
	env := map[string]string{
		"k1": "k1-val",
	}

	type S struct {
		Str1 string `fromenv:"k1"`
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
		nonexported int `fromenv:"k1"`
	}
	var s2 S2
	err = Unmarshal(&s2, Map(map[string]string{"k1": "k1-val"}))
	require.EqualError(t, err, "tag found on unsettable field: field nonexported (int) in struct S2")

	type S3 struct {
		Nonsupported interface{} `fromenv:"k1"`
	}
	var s3 S3
	err = Unmarshal(&s3, Map(map[string]string{"k1": "k1-val"}))
	require.EqualError(t, err, "tag found on unsupported type: field Nonsupported (interface) in struct S3")

	type S4 struct {
		S4Str string `fromenv:"S4Str"`
	}
	type S5 struct {
		S5Str string `fromenv:"S5Str"`
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
		S7str string `fromenv:"S7str,S7default"`
	}
	type S8 struct {
		S8str string `fromenv:"S8str,S8default"`
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
	unsetKeys()
	defer unsetKeys()

	var err error
	type S struct {
		Str1 string `fromenv:"fromenv_test_key1"`
		Str2 string `fromenv:"fromenv_test_key2,str2-def"`
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
