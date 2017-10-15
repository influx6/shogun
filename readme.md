![Shogun](./media/shogun.png)

Shogun
---------

Shogun provides Functions as Commands(FACM), which allows the execution of Go functions from the terminal.
Such functions receive data through the standard input, respond through standard output and standard error
channels of your termianl or pty terminal.

Shogun also creates project files and binaries for all packages, which lets you
quickly generate Go binaries for your functions, which can be moved and run anywhere
individually.

*Inspired by [mage](https://github.com/magefile/mage) and [Amazon Lambda functions](http://docs.aws.amazon.com/lambda/latest/dg/lambda-introduction-function.html) as Runnable items*

*Shogun follows the strict requirement that every information to be received by a function must
come through the standard input file `stdin`, this ensures you can pass arbitrary data in or even
JSON payloads to be loaded into a `Struct` type.*

*More so, all response must either be either an error returned which will be delivered through
the standard error file `stderr` or all functions must receive a `io.WriteCloser` to deliver
response for an execution of a function.*

## Install

```bash
go install -u github.com/influx6/shogun
```

Then run `shogun` to validate successful install:

```bash
> shogun
⠙ Nothing to do...

⡿ Run `shogun help` to see what it takes to make me work.
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
variable. More so, Shogun names all binaries the name of the parent package unless one
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
is executed will be parsed and processed for identification of possible shogun packages,
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

- `func(*Struct) error`
- `func(Context, *Struct) error`
- `func(*Struct, io.WriteCloser) error`
- `func(Context, *Struct, io.WriteCloser) error`

- `func(package.Struct) error`
- `func(Context, package.Struct) error`
- `func(package.Struct, io.WriteCloser) error`
- `func(Context, package.Struct, io.WriteCloser) error`

- `func(*package.Struct) error`
- `func(Context, *package.Struct) error`
- `func(*package.Struct, io.WriteCloser) error`
- `func(Context, *package.Struct, io.WriteCloser) error`

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

func Bob(ctx context.CancelContext, name string) error {
  fmt.Printf("Welcome to bob %q.\n",name)
	return nil
}

func Jija(ctx context.CancelContext, mp ty.Woofer) error {
	return nil
}
```

In shogun, you can tag a function as the default function to be executed every time
when it is called through shogun or through it's generated binary, by tagging it
with a `@default` annotation.


### Using Context

Only the following packages and interfaces are allowed for context usage.
If you need context then it must always be the first argument.

- context "context.Context"
- github.com/influx6/fuax/context "context.CancelContext"

When using `context.Context` package from the internal Go packages, has a means of
timeout for the execution life time of a function. Support of filling context with value
is not planned or desired.

Shogun will use the `-time` flag received through the commandline to set lifetime
timeout for the 2 giving context else the context will not have expiration deadlines.


## CLI Usage

Using the `shogun` command, we can do the following:

- Build shogun based package files
Run this if the shogun files and directories exists right in the root directory.

```bash
shogun build
```

- Build shogun based package files without generating binaries
Run this if the shogun files and directories exists right in the root directory.

```bash
shogun build -skip
```

- Build a shogun based package files in a directory

```bash
shogun build -d=./examples
```

- Build a shogun based package files in a directory and split binaries as single

```bash
shogun build -single -d=./examples -cmd=./cmd
```

Shogun by defaults will be all the first level directories that have shogun files with appropriate
binary names based on package name or if `binaryName` annotation is declared, and will generate
a single binary if there exists any shogun files within the root with subcommands that will
connect to other commands from shogun packages in the first level directories.

This allows you to have a single binary that is bundled with all commands to execution functions
from any other shogun package, but this can be changed to only allow single binaries incase you
want truly separate binaries.

*Note, this only applies, if you have Go files that have the `+build shogun `within the root,
where `shogun build` gets executed.*


- Build a shogun based package files in a directory and store generated packages in directory `cmd`

```bash
shogun build -d=./examples -cmd=./cmd
```

- Build a shogun based package files in a directory without generating binaries

```bash
shogun build -skip -d=./examples
```

- Force rebuild a shogun based package files in a directory

```bash
shogun build -f -d=./examples
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

## Contributions
Contributions are welcome, do please checkout the [Contribution Guidlines](./contrib.md).


Logo is a work of [Shadow Fight Wiki](http://shadowfight.wikia.com/wiki/Characters).
