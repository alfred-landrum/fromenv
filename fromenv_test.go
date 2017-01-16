package fromenv

import (
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func testLooker(m map[string]string) func(*config) error {
	f := func(key string) (val string, ok bool) {
		val, ok = m[key]
		return
	}
	return LookupEnv(f)
}

func testNilLooker() func(*config) error {
	f := func(string) (string, bool) {
		panic("unexpected lookup in test")
	}
	return LookupEnv(f)
}

func TestConfig(t *testing.T) {
	t.Parallel()

	var err error
	type S1 struct {
		Str1 string `fromenv:"k1"`
	}
	var s1 S1

	badconf := func(c *config) error {
		return errors.New("config error")
	}
	err = Configure(&s1, badconf)
	require.EqualError(t, err, "config error")
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

	err = Configure(&s1, testLooker(env))
	require.NoError(t, err)
	require.Equal(t, s1.Str1, "k1-val")
}

func TestTypeLogic(t *testing.T) {
	t.Parallel()

	var err error
	err = Configure(nil, testNilLooker())
	require.EqualError(t, err, "passed non-pointer or nil pointer")

	type S0 struct{}
	var s0 S0
	err = Configure(s0, testNilLooker())
	require.EqualError(t, err, "passed non-pointer or nil pointer")

	type S1 struct {
		notag int
	}
	var s1 S1
	err = Configure(&s1, testNilLooker())
	require.NoError(t, err)

	type S2 struct {
		nonexported int `fromenv:"k1"`
	}
	var s2 S2
	err = Configure(&s2, testNilLooker())
	require.EqualError(t, err, "tag found on unsettable field: field nonexported (int) in struct S2")

	type S3 struct {
		Nonsupported interface{} `fromenv:"k1"`
	}
	var s3 S3
	err = Configure(&s3, testNilLooker())
	require.EqualError(t, err, "tag found on unsupported type: field Nonsupported (interface) in struct S3")

	type S4 struct {
		S4Str string `fromenv:"S4Str-val"`
	}
	type S5 struct {
		S5Str string `fromenv:"S5Str-val"`
	}
	type S6 struct {
		S4       S4
		S5ptr    *S5
		S5nilptr *S5
	}
	env6 := map[string]string{
		"S4Int": "S4Int-val",
		"S5Int": "S5Int-val",
	}
	s6 := S6{
		S5nilptr: nil,
		S5ptr:    &S5{},
	}
	err = Configure(&s6, testLooker(env6))
	require.NoError(t, err)
	require.Equal(t, env6["S4Str-val"], s6.S4.S4Str)
	require.Equal(t, env6["S5Str-val"], s6.S5ptr.S5Str)
}

func TestString(t *testing.T) {
	t.Parallel()

	env := map[string]string{
		"k1": "k1-val",
	}

	type S1 struct {
		Str1 string `fromenv:"k1"`
	}

	var s1 S1
	err := Configure(&s1, testLooker(env))
	require.NoError(t, err)
	require.Equal(t, s1.Str1, "k1-val")

	type S2 struct {
		Str1 string `fromenv:"k1,not-used-default"`
	}

	var s2 S2
	err = Configure(&s2, testLooker(env))
	require.NoError(t, err)
	require.Equal(t, s2.Str1, "k1-val")

	type S3 struct {
		Str1 string `fromenv:"nokey,def-val,with-comma"`
	}

	var s3 S3
	err = Configure(&s3, testLooker(env))
	require.NoError(t, err)
	require.Equal(t, s3.Str1, "def-val,with-comma")
}

func TestInt(t *testing.T) {
	t.Parallel()

	env := map[string]string{
		"k1": "0",
		"k2": "i-am-not-an-int",
	}

	type S1 struct {
		Int1 int `fromenv:"k1"`
	}

	var s1 S1
	err := Configure(&s1, testLooker(env))
	require.NoError(t, err)
	require.Equal(t, s1.Int1, 0)

	type S2 struct {
		Int2 int `fromenv:"k2"`
	}

	var s2 S2
	err = Configure(&s2, testLooker(env))
	require.Contains(t, err.Error(), "failed to configure from k2")
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
	err := Configure(&s1, testLooker(env))
	require.NoError(t, err)
	require.Equal(t, s1.Bool1, true)

	type S2 struct {
		Bool2 bool `fromenv:"k2"`
	}

	var s2 S2
	err = Configure(&s2, testLooker(env))
	require.Contains(t, err.Error(), "failed to configure from k2")
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
	err = Configure(&s1)
	require.NoError(t, err)
	require.Equal(t, s1.Str1, "")
	require.Equal(t, s1.Str2, "str2-def")

	unsetKeys()
	os.Setenv("fromenv_test_key1", "key1-value")
	os.Setenv("fromenv_test_key2", "key2-value")
	var s2 S
	err = Configure(&s2)
	require.NoError(t, err)
	require.Equal(t, s2.Str1, "key1-value")
	require.Equal(t, s2.Str2, "key2-value")
}
