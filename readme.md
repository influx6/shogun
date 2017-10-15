![Shogun](./media/shogun.png)

Shogun
---------

Shogun provides Functions as Commands(FACM), which allows the execution of Go functions from commandline,
through the standard input, output and error channels of your terminal or pty terminal.
Shogun also creates project files and binaries for all packages marked which also lets
you quickly create CLI binaries for your functions, which can be moved and run anywhere
individually.

*Inspired by [mage](https://github.com/magefile/mage) and [Amazon Lambda functions](http://docs.aws.amazon.com/lambda/latest/dg/lambda-introduction-function.html) as Runnable items*

## Install

```bash
go install -u github.com/influx6/shogun
```

## Writing Shogun Packages

Writing Go package which are to be used by Shogun to generate binaries are rather simple,
and only require that each package has all it files tagged with the following build tag above
it's package declaration with space in between:

```
// +build shogun

package something
```

Your are free to use any other build tag as well and will be sorted accordingly.

Shogun by default will save binaries into the `GOBIN` or `GOPATH/bin` path extracted
from the environment, however this can be changed by setting a `SHOGUNBIN` environment
varable. More so, Shogun names all binaries the name of the parent package unless one
declares an explicit annotation `@binaryName` at the package level.

```
// +build shogun

// Package do does something.
//
//@binaryName(name => shogunate_bin)
package do
```

If you wished to add a description for the binary command, we can add a `desc` attribute
in a json block of the `@binaryName` annotation.

```
// +build shogun

/* Package do does something.

@binaryName(asJSON, name => shogunate_bin, {
  {
    "desc": "shogunate_bin provides a nice means"
  }
})
*/
package do
```

All binaries created by shogun are self complete and can equally be called directly without
the `shogun` command, which makes it very usable for easy deployable self contained executables
that can be used in place where behaviors need to be exposed as functions.

Shogun packages are normal Go packages and all directories within the root where shogun
is runned will be parsed and processed for identification of possible shogun packages,
where those identified will each package will be a binary in and of itself and the main
package if any found will combine all other binaries into a single one if so desired.


### Writing Functions for Shogun

Shogun focuses on the execution of functions, that supports a limited set of formats,
More so, to match needs of most with `Context` objects, the function formats support
the usage of Context as first place arguments.

-	`func()`
- `func() error`

- `func(Context)`
- `func(Context) error`

- `func(map[string]interface{}) error`
- `func(Context, map[string]interface{}) error`

- `func(Struct) error`
- `func(Context, Struct) error`
- `func(Struct, io.WriteCloser) error`
- `func(Context, Struct, io.WriteCloser) error`

- `func(package.Type) error`
- `func(Context, package.Type) error`
- `func(Struct, io.WriteCloser) error`
- `func(Context, package.Type, io.WriteCloser) error`

- `func(io.Reader) error`
- `func(io.Reader, io.WriteCloser) error`
- `func(Context, io.Reader, io.WriteCloser) error`

*Where `Context` => represents the context package used of the 3 allowed.*
*Where `Struct`   => represents any struct declared in package*
*where `Interface` => represents any interface declared in package*
*where `package.Type` => represents Struct type imported from other package*

*Any other thing beyond this type formats won't be allowed and will be ignored in
function list and execution.*

```go
// +build shogun

// Package katanas provides exported functions as tasks runnable from commandline.
//
// @binaryName(name => katana-shell)
//
package katanas

import (
	"fmt"
	"io"

	"github.com/influx6/faux/context"
	ty "github.com/influx6/shogun/examples/types"
)

type wondra struct {
	Name string
}

func Draw() {}

// Slash is the default tasks due to below annotation.
// @default
func Slash() error {
	fmt.Println("Welcome to Katana slash!")
	return nil
}

// Buba is bub.
func Buba(ctx context.ValueBagContext) {
}

func Bob(ctx context.CancelContext) error {
	return nil
}

func Jija(ctx context.CancelContext, mp ty.Woofer) error {
	return nil
}
```

In shogun, you can tag a function as the default function to be runned by using
the  `@default` annotation. This ensures if binary generated is called or if shogun
command is called with binary name without argument, then that function will be called.


### Using Context

Only the following packages and interfaces are allowed for context usage.
If you need context then it must always be the first argument.

- context "context.Context"
- github.com/influx6/fuax/context "context.CancelContext"

When using `context.Context` as the context type which is part of the Go core packages,
as far as the context is the only argument of any function if any json sent as input,
then all json key-value pairs will be copied into the context.

Shogun will use the `-time` flag to set lifetime timeout for the 2 giving context else
the context will not have expiration deadlines.


## CLI Usage

Using the `shogun` command, we can do the following:

- Build a package shogun files

```bash
shogun build
```

Shogun will hash shogun files and ensure only when changes occur will a new build be
made and binary will be stored in binary location as dictated by environment
variable `SHOGUNBIN` or default `GOBIN`/`GOPATH/bin` .

- List all functions with

```bash
shogun list
```

- List all functions with short commentary

```bash
shogun help {{BINARYNAME}} {{FunctionName}}
```

- List all functions with full commentary and source

```bash
shogun help -s {{BINARYNAME}} {{FunctionName}}
```

- Run function of package binary expecting no input

```bash
shogun {{BINARYNAME}} {{FUNCTIONNAME}}
```

- Run function of package binary with standard input

```bash
echo "We lost the war" | shogun {{BINARYNAME}} {{FUNCTIONNAME}}
```

- Run function of package binary shogun files with json input

```bash
{"name":"bat"} | shogun {{BINARYNAME}} {{FUNCTIONNAME}} 
```
