package fromenv

import (
	"fmt"
	"net/url"
	"os"
	"time"
)

func Example() {
	var c struct {
		Field1 string `env:"KEY1"`
	}
	os.Setenv("KEY1", "foo")
	_ = Unmarshal(&c)
	fmt.Println(c.Field1)
	// Ouput: foo
}

func Example_default() {
	var c struct {
		Field1 string `env:"KEY1=key1default"`
	}

	os.Unsetenv("KEY1")

	_ = Unmarshal(&c)
	fmt.Println(c.Field1)
	// Output: key1default
}

func Example_inner() {
	var c struct {
		Inner struct {
			Field2 string `env:"KEY2"`
		}
	}

	os.Setenv("KEY2", "inner too")

	_ = Unmarshal(&c)
	fmt.Println(c.Inner.Field2)
	// Output: inner too
}

func ExampleSetFunc() {
	durSetter := func(t *time.Duration, s string) error {
		x, err := time.ParseDuration(s)
		*t = x
		return err
	}

	urlSetter := func(u *url.URL, s string) error {
		x, err := url.Parse(s)
		*u = *x
		return err
	}

	type config struct {
		Timeout time.Duration `env:"GAP=1000ms"`
		Server  *url.URL      `env:"PLACE=http://www.github.com"`
	}

	var c config
	_ = Unmarshal(&c, SetFunc(durSetter), SetFunc(urlSetter))
	fmt.Println(c.Timeout, c.Server.Hostname())
	// Output: 1s www.github.com
}
