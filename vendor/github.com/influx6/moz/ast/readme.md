AST
-------
Ast provides the ability to distill go packages and files into individual structures
that better represent their type, data and content. It provides a cleaner and more
usage structures that extra as much details from the Go ast so you don't have to.

This structures then make it possible to transform existing code structures as desired,
more so, ast targets annotation based types, where Go types such has interfaces, structs,
function and other types can be annotated with markers using the `@` prefix.

By relying on this annotation, AST then lets you provide functions to generate new code or
content to be written to file has dictated by you. It is barebones but very flexible for
creating custom code generation possibilities.

Annotation Code Generation
----------------------------

Moz provides a annotation style code generation system apart from it's code generation structures. This is provide to allow descriptive annotations to be added to giving Go structures (`interface`, `struct`, `type alises`) within their comments and as well as to the package.

This annotation then are passed by the moz `annotation` CLI tooling which can generate a series of files and packages based on the internal logic of the generator associated with that annotation to meet the needs for that type.

For example: If we wanted to be able to generate code for database CRUD activities without having to use ORMs or write such code manually, with the Moz annotation code generation ability, we can create a `struct` generator that can use a `@mongo` annotation, which generates mongo CRUD functions which expect such a type and perform the appropriate CRUD operations.

See the [Example](../examples/) directory, which demonstrates use of annotations to code generate other parts of a project or mock up implementation detail for an interface using annotations.

### AST Annotation Functions
AST provides 4 types of Annotation generators, which are function types which provide the necessary operations to be performed to create the underline series of sources to be generated for each annotation. More so, these functions all receiving a `string` has their first argument, which is the relative path of a directory (existing/not-existing) that whatever content to be written will be created into. This allows the functions to be aware of path changes as needed in the contents they may generate.

AST provide the following generators type functions:

#### StructType Code Generators

These functions types are used to provide code generation instructions for Go type declarations and are the ones who define the end result of
what an annotation produces.

_See [Annotations](./annotations) for code samples of different annotation functions._

```go
type StructAnnotationGenerator func(string, AnnotationDeclaration, StructDeclaration, PackageDeclaration, Package) ([]gen.WriteDirective, error)
```

*This function is expected to return a slice of `WriteDirective` which contains file name, `WriterTo` object and a possible `Dir` relative path which the contents should be written to.*

#### InterfaceType Code Generators

This functions are specific to provide code generation instructions for interface declarations which the given annotation is attached to.

```go
type InterfaceAnnotationGenerator func(string,AnnotationDeclaration, InterfaceDeclaration, PackageDeclaration, Package) ([]gen.WriteDirective, error)
```

*This function is expected to return a slice of `WriteDirective` which contains file name, `WriterTo` object and a possible `Dir` relative path which the contents should be written to.*

#### PackageType Code Generators

This functions are specific to provide code generation instructions for given annotation declared on the package comment block.

```go
type PackageAnnotationGenerator func(string, AnnotationDeclaration, PackageDeclaration, Package) ([]gen.WriteDirective, error)
```

*This function is expected to return a slice of `WriteDirective` which contains file name, `WriterTo` object and a possible `Dir` relative path which the contents should be written to.*

#### Non(Struct|Interface) Code Generators

This functions are specific to provide code generation instructions for non-struct and non-interface declarations which the given annotation is attached to.

```go
type TypeAnnotationGenerator func(string, AnnotationDeclaration, TypeDeclaration, PackageDeclaration, Package) ([]gen.WriteDirective, error)
```

*This function is expected to return a slice of `WriteDirective` which contains file name, `WriterTo` object and a possible `Dir` relative path which the contents should be written to.*


Example
------------

#### Generate code structures from an interface

1. Create a file and add the following contents defining a interface we wish to
create it's implementation structures by annotating with a `@iface` annotation.

```go
package mock

//go:generate moz generate

import (
	"io"

	toml "github.com/BurntSushi/toml"
)

// MofInitable defines a interface for a Mof.
// @iface
type MofInitable interface {
	Ignitable
	Crunch() (cr string)
	Configuration() toml.Primitive
	Location(string) (GPSLoc, error)
	WriterTo(io.Writer) (int64, error)
	Maps(string) (map[string]GPSLoc, error)
	MapsIn(string) (map[string]*GPSLoc, error)
	MapsOut(string) (map[*GPSLoc]string, error)
	Drop() (*GPSLoc, *toml.Primitive, *[]byte, *[5]byte)
	Close() (chan struct{}, chan toml.Primitive, chan string, chan []byte, chan *[]string)
	Bob() chan chan struct{}
}


// Ignitable defines a struct which is used to ignite the package.
type Ignitable interface {
	Ignite() string
}

// GPSLoc defines a struct to hold long and lat values for a gps location.
type GPSLoc struct {
	Lat  float64
	Long float64
}

```

2. Navigate to where file is stored (We assume it's in your GOPATH) and run

```
go generate
```

or

```
moz generate
```

The command above will generate all necessary files and packages ready for editing.

See [Mock Example](../examples/mock) for end result.
