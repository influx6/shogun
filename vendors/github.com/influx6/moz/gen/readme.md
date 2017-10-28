Gen
--------
Gen is a package that provides code generation using functional compoistion of functions, where
each is built around the concept of composable `io.WriterTo`. Each writer wraps the passed content of
the next writer creating flexible structures for generating code or text easily.

Code Generation Structures
---------------------------

This package provides sets of structures which define specific code structures and are used to built a programmatically combination that define the expected code to be produced. It also provides a functional composition style functions that provide a cleaner and more descriptive approach to how these blocks are combined.

The code gen is heavily geared towards the use of `text/template` but also ensures to be flexible to provide non-template based structures that work as well.

### Example

- Generate a struct with moz

```go
import "github.com/influx6/moz/gen"

floppy := gen.Struct(
		gen.Name("Floppy"),
		gen.Commentary(
			gen.Text("Floppy provides a basic function."),
			gen.Text("Demonstration of using floppy API."),
		),
		gen.Annotations(
			"Flipo",
			"API",
		),
		gen.Field(
			gen.Name("Name"),
			gen.Type("string"),
			gen.Tag("json", "name"),
		),
)

var source bytes.Buffer

floppy.WriteTo(&source) /*
// Floppy provides a basic function.
//
// Demonstration of using floppy API.
//
//
//@Flipo
//@API
type Floppy struct {

    Name string `json:"name"`

}
*/
```


- Generate a function with moz

```go
import "github.com/influx6/moz/gen"

main := gen.Function(
    gen.Name("main"),
    gen.Constructor(
        gen.FieldType(
            gen.Name("v"),
            gen.Type("int"),
        ),
        gen.FieldType(
            gen.Name("m"),
            gen.Type("string"),
        ),
    ),
    gen.Returns(),
    gen.SourceText(`	fmt.Printf("Welcome to Lola Land");`, nil),
)

var source bytes.Buffer

main.WriteTo(&source) /*
func main(v int, m string) {
	fmt.Printf("Welcome to Lola Land");
}
*/
```
