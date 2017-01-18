

# fromenv
`import "github.com/alfred-landrum/fromenv"`

* [Overview](#pkg-overview)
* [Index](#pkg-index)

## <a name="pkg-overview">Overview</a>
Package fromenv can set specially tagged struct fields with values
from the environment.


	var c struct {
		Field1 string  	`fromenv:"FIELD1_KEY,my-default"`
		Field2 int     	`fromenv:"FIELD2_KEY,7"`
		Field3 bool    	`fromenv:"FIELD3_KEY"`
	
		Inner struct {
			Field4 string	`fromenv:"FIELD4_KEY"`
		}
	}
	
	os.Setenv("FIELD1_KEY","foo")
	os.Unsetenv("FIELD2_KEY")
	os.Setenv("FIELD3_KEY","true")
	os.Setenv("FIELD4_KEY","inner too!")
	
	err := fromenv.Unmarshal(&c)
	// c.Field1 == "foo"
	// c.Field2 == 7
	// c.Field3 == true
	// c.Inner.Field4 == "inner too!"




## <a name="pkg-index">Index</a>
* [func Unmarshal(in interface{}, options ...Option) error](#Unmarshal)
* [type LookupEnvFunc](#LookupEnvFunc)
* [type Option](#Option)
  * [func DefaultsOnly() Option](#DefaultsOnly)
  * [func Looker(f LookupEnvFunc) Option](#Looker)
  * [func Map(m map[string]string) Option](#Map)
* [type URL](#URL)
  * [func (u *URL) Set(s string) error](#URL.Set)
  * [func (u *URL) String() string](#URL.String)


#### <a name="pkg-files">Package files</a>
[fromenv.go](/src/github.com/alfred-landrum/fromenv/fromenv.go) [types.go](/src/github.com/alfred-landrum/fromenv/types.go) 





## <a name="Unmarshal">func</a> [Unmarshal](/src/target/fromenv.go?s=1521:1576#L48)
``` go
func Unmarshal(in interface{}, options ...Option) error
```
Unmarshal takes a pointer to a struct, recursively looks for struct
fields with a "fromenv" tag, and sets the field to the value of the
environment variable given in the tag. A fromenv tag may optionally
specify a default value; the field will be set to this value if the
environment variable is not present.

By default, the "os.LookupEnv" function is used to find the value
for an environment variable.

Basic types supported are: string, bool, int, uint8, uint16, uint32,
uint64, int, int8, int16, int32, int64, float32, float64.

The flag package's Value interface is also supported.




## <a name="LookupEnvFunc">type</a> [LookupEnvFunc](/src/target/fromenv.go?s=2805:2867#L97)
``` go
type LookupEnvFunc func(key string) (value *string, err error)
```
A LookupEnvFunc retrieves the value of the environment variable
named by the key. If the variable isn't present, a nil pointer
is returned.










## <a name="Option">type</a> [Option](/src/target/fromenv.go?s=3092:3117#L108)
``` go
type Option func(*config)
```
An Option is a functional option for Unmarshal.







### <a name="DefaultsOnly">func</a> [DefaultsOnly](/src/target/fromenv.go?s=2527:2553#L86)
``` go
func DefaultsOnly() Option
```
DefaultsOnly configures Unmarshal to only set fields with a tag-defined
default to that default, ignoring other fields and the environment.


### <a name="Looker">func</a> [Looker](/src/target/fromenv.go?s=2956:2991#L101)
``` go
func Looker(f LookupEnvFunc) Option
```
Looker configures the environment lookup function used during an
Unmarshal call.


### <a name="Map">func</a> [Map](/src/target/fromenv.go?s=2190:2226#L73)
``` go
func Map(m map[string]string) Option
```
Map configures Unmarshal to use the given map for environment lookups.





## <a name="URL">type</a> [URL](/src/target/types.go?s=310:326#L4)
``` go
type URL url.URL
```
URL merely provides a net/url.URL wrapper that matches the flag.Value
interface for ease of using with fromenv.










### <a name="URL.Set">func</a> (\*URL) [Set](/src/target/types.go?s=452:485#L8)
``` go
func (u *URL) Set(s string) error
```
Set calls url.ParseRequestURI on the input string; on success, this
instance is set to the parsed net/url.URL result.




### <a name="URL.String">func</a> (\*URL) [String](/src/target/types.go?s=586:615#L17)
``` go
func (u *URL) String() string
```







- - -
Generated by [godoc2md](http://godoc.org/github.com/davecheney/godoc2md)
