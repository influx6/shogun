package ast

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/influx6/faux/metrics"
	"github.com/influx6/moz/gen"
)

// TypeAnnotationGenerator defines a function which generates specific code related to the giving
// Annotation for a non-struct, non-interface type declaration. This allows you to apply and create
// new sources specifically for a giving type(non-struct, non-interface).
// It is responsible to fully contain all operations required to both generator any source and write such to
type TypeAnnotationGenerator func(string, AnnotationDeclaration, TypeDeclaration, PackageDeclaration, Package) ([]gen.WriteDirective, error)

// FunctionAnnotationGenerator defines a function which generates specific code related to the giving
// Annotation. This allows you to generate a new source file containg source code for a giving struct type.
// It is responsible to fully contain all operations required to both generator any source and write such to.
type FunctionAnnotationGenerator func(string, AnnotationDeclaration, FuncDeclaration, PackageDeclaration, Package) ([]gen.WriteDirective, error)

// StructAnnotationGenerator defines a function which generates specific code related to the giving
// Annotation. This allows you to generate a new source file containg source code for a giving struct type.
// It is responsible to fully contain all operations required to both generator any source and write such to.
type StructAnnotationGenerator func(string, AnnotationDeclaration, StructDeclaration, PackageDeclaration, Package) ([]gen.WriteDirective, error)

// InterfaceAnnotationGenerator defines a function which generates specific code related to the giving
// Annotation. This allows you to generate a new source file containg source code for a giving interface type.
// It is responsible to fully contain all operations required to both generator any source and write such to
// appropriate files as intended, meta-data about package, and file paths are already include in the PackageDeclaration.
type InterfaceAnnotationGenerator func(string, AnnotationDeclaration, InterfaceDeclaration, PackageDeclaration, Package) ([]gen.WriteDirective, error)

// PackageAnnotationGenerator defines a function which generates specific code related to the giving
// Annotation for a package. This allows you to apply and create new sources specifically because of a
// package wide annotation.
// It is responsible to fully contain all operations required to both generator any source and write such to
// All generators are expected to return
type PackageAnnotationGenerator func(string, AnnotationDeclaration, PackageDeclaration, Package) ([]gen.WriteDirective, error)

//===========================================================================================================

// Annotations defines a struct which contains a map of all annotation code generator.
type Annotations struct {
	Types      map[string]TypeAnnotationGenerator
	Structs    map[string]StructAnnotationGenerator
	Functions  map[string]FunctionAnnotationGenerator
	Packages   map[string]PackageAnnotationGenerator
	Interfaces map[string]InterfaceAnnotationGenerator
}

// AnnotationRegistry defines a structure which contains giving list of possible
// annotation generators for both package level and type level declaration.
type AnnotationRegistry struct {
	metrics              metrics.Metrics
	ml                   sync.RWMutex
	typeAnnotations      map[string]TypeAnnotationGenerator
	structAnnotations    map[string]StructAnnotationGenerator
	pkgAnnotations       map[string]PackageAnnotationGenerator
	interfaceAnnotations map[string]InterfaceAnnotationGenerator
	functionAnnotations  map[string]FunctionAnnotationGenerator
}

// NewAnnotationRegistry returns a new instance of a AnnotationRegistry.
func NewAnnotationRegistry() *AnnotationRegistry {
	return &AnnotationRegistry{
		metrics:              metrics.New(),
		typeAnnotations:      make(map[string]TypeAnnotationGenerator),
		structAnnotations:    make(map[string]StructAnnotationGenerator),
		pkgAnnotations:       make(map[string]PackageAnnotationGenerator),
		interfaceAnnotations: make(map[string]InterfaceAnnotationGenerator),
		functionAnnotations:  make(map[string]FunctionAnnotationGenerator),
	}
}

// NewAnnotationRegistryWith returns a new instance of a AnnotationRegistry.
func NewAnnotationRegistryWith(log metrics.Metrics) *AnnotationRegistry {
	return &AnnotationRegistry{
		metrics:              log,
		typeAnnotations:      make(map[string]TypeAnnotationGenerator),
		structAnnotations:    make(map[string]StructAnnotationGenerator),
		pkgAnnotations:       make(map[string]PackageAnnotationGenerator),
		interfaceAnnotations: make(map[string]InterfaceAnnotationGenerator),
		functionAnnotations:  make(map[string]FunctionAnnotationGenerator),
	}
}

// Clone returns a type which contains all copies of the generators provided by
// the AnnotationRegistry.
func (a *AnnotationRegistry) Clone() Annotations {
	a.ml.RLock()
	defer a.ml.RUnlock()

	var cloned Annotations
	cloned.Types = make(map[string]TypeAnnotationGenerator)
	cloned.Structs = make(map[string]StructAnnotationGenerator)
	cloned.Packages = make(map[string]PackageAnnotationGenerator)
	cloned.Interfaces = make(map[string]InterfaceAnnotationGenerator)
	cloned.Functions = make(map[string]FunctionAnnotationGenerator)

	for name, item := range a.pkgAnnotations {
		cloned.Packages[name] = item
	}

	for name, item := range a.functionAnnotations {
		cloned.Functions[name] = item
	}

	for name, item := range a.structAnnotations {
		cloned.Structs[name] = item
	}

	for name, item := range a.typeAnnotations {
		cloned.Types[name] = item
	}

	for name, item := range a.interfaceAnnotations {
		cloned.Interfaces[name] = item
	}

	return cloned
}

// CopyStrategy defines a int type used to represent a copy strategy for
// cloning a AnnotationStrategy.
type CopyStrategy int

// Contains different copy strategy.
const (
	OursOverTheirs CopyStrategy = iota + 1
	TheirsOverOurs
)

// Copy copies over all available type generators from the provided AnnotationRegistry with
// the CopyStrategy.
func (a *AnnotationRegistry) Copy(registry *AnnotationRegistry, strategy CopyStrategy) {
	cloned := registry.Clone()

	a.ml.Lock()
	defer a.ml.Unlock()

	for name, item := range cloned.Packages {
		_, ok := a.pkgAnnotations[name]

		if !ok || (ok && strategy == TheirsOverOurs) {
			a.pkgAnnotations[name] = item
		}
	}

	for name, item := range cloned.Functions {
		_, ok := a.functionAnnotations[name]
		if !ok || (ok && strategy == TheirsOverOurs) {
			a.functionAnnotations[name] = item
		}
	}

	for name, item := range cloned.Types {
		_, ok := a.typeAnnotations[name]
		if !ok || (ok && strategy == TheirsOverOurs) {
			a.typeAnnotations[name] = item
		}
	}

	for name, item := range cloned.Structs {
		_, ok := a.structAnnotations[name]
		if !ok || (ok && strategy == TheirsOverOurs) {
			a.structAnnotations[name] = item
		}
	}

	for name, item := range cloned.Interfaces {
		_, ok := a.interfaceAnnotations[name]
		if !ok || (ok && strategy == TheirsOverOurs) {
			a.interfaceAnnotations[name] = item
		}
	}
}

// MustPackage returns the annotation generator associated with the giving annotation name.
func (a *AnnotationRegistry) MustPackage(annotation string) PackageAnnotationGenerator {
	annon, err := a.GetPackage(annotation)
	if err == nil {
		return annon
	}

	panic(err)
}

// AnnotationWriteDirective defines a type which provides a WriteDiretive and the associated
// name.
type AnnotationWriteDirective struct {
	gen.WriteDirective
	Annotation string
}

// ParseDeclr runs the generators suited for each declaration and type returning a slice of
// Annotationgen.WriteDirective that delivers the content to be created for each piece.
func (a *AnnotationRegistry) ParseDeclr(pkg Package, declr PackageDeclaration, toDir string) ([]AnnotationWriteDirective, error) {
	var directives []AnnotationWriteDirective

	// Generate directives for package level
	for _, annotation := range declr.Annotations {
		a.metrics.Emit(metrics.Info("Directive Generation"),
			metrics.With("Level", "Package"), metrics.With("Annotaton", annotation.Name), metrics.With("Params", annotation.Params), metrics.With("Arguments", annotation.Arguments), metrics.With("Template", annotation.Template))

		generator, err := a.GetPackage(annotation.Name)
		if err != nil {
			continue
		}

		drs, err := generator(toDir, annotation, declr, pkg)
		if err != nil {
			a.metrics.Emit(metrics.Error(errors.New("Directive Generation")),
				metrics.With("error", err), metrics.With("Level", "Package"), metrics.With("Annotaton", annotation.Name), metrics.With("Params", annotation.Params), metrics.With("Arguments", annotation.Arguments), metrics.With("Template", annotation.Template))
			return nil, err
		}

		a.metrics.Emit(metrics.Info("Directive Generation: Success"),
			metrics.With("Level", "Package"),
			metrics.With("Directive", len(drs)),
			metrics.With("Annotaton", annotation.Name),
			metrics.With("Params", annotation.Params),
			metrics.With("Arguments", annotation.Arguments),
			metrics.With("Template", annotation.Template))

		for _, directive := range drs {
			directives = append(directives, AnnotationWriteDirective{
				WriteDirective: directive,
				Annotation:     annotation.Name,
			})
		}
	}

	for _, inter := range declr.Interfaces {
		for _, annotation := range inter.Annotations {
			a.metrics.Emit(metrics.Info("Directive Generation"),
				metrics.With("Level", "Interface"),
				metrics.With("Annotaton", annotation.Name),
				metrics.With("Interface", inter.Object.Name.Name),
				metrics.With("Params", annotation.Params),
				metrics.With("Arguments", annotation.Arguments),
				metrics.With("Template", annotation.Template))

			generator, err := a.GetInterfaceType(annotation.Name)
			if err != nil {
				a.metrics.Emit(metrics.Error(errors.New("Directive Generation")),
					metrics.With("error", err),
					metrics.With("Level", "Interface"),
					metrics.With("Annotaton", annotation.Name),
					metrics.With("Interface", inter.Object.Name.Name),
					metrics.With("Params", annotation.Params),
					metrics.With("Arguments", annotation.Arguments),
					metrics.With("Template", annotation.Template))
				continue
			}

			drs, err := generator(toDir, annotation, inter, declr, pkg)
			if err != nil {
				a.metrics.Emit(metrics.Error(errors.New("Directive Generation")),
					metrics.With("error", err),
					metrics.With("Level", "Interface"),
					metrics.With("Annotaton", annotation.Name),
					metrics.With("Interface", inter.Object.Name.Name),
					metrics.With("Params", annotation.Params),
					metrics.With("Arguments", annotation.Arguments),
					metrics.With("Template", annotation.Template))
				return nil, err
			}

			a.metrics.Emit(metrics.Info("Directive Generation: Success"),
				metrics.With("Level", "Struct"),
				metrics.With("Directive", len(drs)),
				metrics.With("Annotaton", annotation.Name),
				metrics.With("nterface", inter.Object.Name.Name),
				metrics.With("Params", annotation.Params),
				metrics.With("Arguments", annotation.Arguments),
				metrics.With("Template", annotation.Template))

			for _, directive := range drs {
				directives = append(directives, AnnotationWriteDirective{
					WriteDirective: directive,
					Annotation:     annotation.Name,
				})
			}
		}
	}

	for _, structs := range declr.Structs {
		for _, annotation := range structs.Annotations {
			a.metrics.Emit(metrics.Info("Directive Generation"),
				metrics.With("Level", "Struct"),
				metrics.With("Annotaton", annotation.Name),
				metrics.With("Struct", structs.Object.Name.Name),
				metrics.With("Params", annotation.Params),
				metrics.With("Arguments", annotation.Arguments),
				metrics.With("Template", annotation.Template))

			generator, err := a.GetStructType(annotation.Name)
			if err != nil {
				a.metrics.Emit(metrics.Error(errors.New("Directive Generation")),
					metrics.With("error", err),
					metrics.With("Level", "Struct"),
					metrics.With("Annotaton", annotation.Name),
					metrics.With("Struct", structs.Object.Name.Name),
					metrics.With("Params", annotation.Params),
					metrics.With("Arguments", annotation.Arguments),
					metrics.With("Template", annotation.Template))
				continue
			}

			drs, err := generator(toDir, annotation, structs, declr, pkg)
			if err != nil {
				a.metrics.Emit(metrics.Error(errors.New("Directive Generation")),
					metrics.With("error", err),
					metrics.With("Level", "Struct"),
					metrics.With("Annotaton", annotation.Name),
					metrics.With("Struct", structs.Object.Name.Name),
					metrics.With("Params", annotation.Params),
					metrics.With("Arguments", annotation.Arguments),
					metrics.With("Template", annotation.Template))
				return nil, err
			}

			a.metrics.Emit(metrics.Info("Directive Generation: Success"),
				metrics.With("Level", "Struct"),
				metrics.With("Directive", len(drs)),
				metrics.With("Annotaton", annotation.Name),
				metrics.With("Struct", structs.Object.Name.Name),
				metrics.With("Params", annotation.Params),
				metrics.With("Arguments", annotation.Arguments),
				metrics.With("Template", annotation.Template))

			for _, directive := range drs {
				directives = append(directives, AnnotationWriteDirective{
					WriteDirective: directive,
					Annotation:     annotation.Name,
				})
			}
		}
	}

	for _, typ := range declr.Functions {
		for _, annotation := range typ.Annotations {
			a.metrics.Emit(metrics.Info("Directive Generation"),
				metrics.With("Level", "Type"),
				metrics.With("Annotaton", annotation.Name),
				metrics.With("Function", typ.FuncDeclr.Name.Name),
				metrics.With("Params", annotation.Params),
				metrics.With("Arguments", annotation.Arguments),
				metrics.With("Template", annotation.Template))

			generator, err := a.GetFunctionType(annotation.Name)
			if err != nil {
				a.metrics.Emit(metrics.Error(errors.New("Directive Generation")),
					metrics.With("error", err),
					metrics.With("Level", "Type"),
					metrics.With("Annotaton", annotation.Name),
					metrics.With("Function", typ.FuncDeclr.Name.Name),
					metrics.With("Params", annotation.Params),
					metrics.With("Arguments", annotation.Arguments),
					metrics.With("Template", annotation.Template))
				continue
			}

			drs, err := generator(toDir, annotation, typ, declr, pkg)
			if err != nil {
				a.metrics.Emit(metrics.Error(errors.New("Directive Generation")),
					metrics.With("error", err),
					metrics.With("Level", "Type"),
					metrics.With("Annotaton", annotation.Name),
					metrics.With("Function", typ.FuncDeclr.Name.Name),
					metrics.With("Params", annotation.Params),
					metrics.With("Arguments", annotation.Arguments),
					metrics.With("Template", annotation.Template))
				return nil, err
			}

			a.metrics.Emit(metrics.Info("Directive Generation: Success"),
				metrics.With("Level", "Type"),
				metrics.With("Directive", len(drs)),
				metrics.With("Annotaton", annotation.Name),
				metrics.With("Function", typ.FuncDeclr.Name.Name),
				metrics.With("Params", annotation.Params),
				metrics.With("Arguments", annotation.Arguments),
				metrics.With("Template", annotation.Template))

			for _, directive := range drs {
				directives = append(directives, AnnotationWriteDirective{
					WriteDirective: directive,
					Annotation:     annotation.Name,
				})
			}
		}
	}

	for _, typ := range declr.Types {
		for _, annotation := range typ.Annotations {
			a.metrics.Emit(metrics.Info("Directive Generation"),
				metrics.With("Level", "Type"),
				metrics.With("Annotaton", annotation.Name),
				metrics.With("Type", typ.Object.Name.Name),
				metrics.With("Params", annotation.Params),
				metrics.With("Arguments", annotation.Arguments),
				metrics.With("Template", annotation.Template))

			generator, err := a.GetType(annotation.Name)
			if err != nil {
				a.metrics.Emit(metrics.Error(errors.New("Directive Generation")),
					metrics.With("error", err),
					metrics.With("Level", "Type"),
					metrics.With("Annotaton", annotation.Name),
					metrics.With("Type", typ.Object.Name.Name),
					metrics.With("Params", annotation.Params),
					metrics.With("Arguments", annotation.Arguments),
					metrics.With("Template", annotation.Template))
				continue
			}

			drs, err := generator(toDir, annotation, typ, declr, pkg)
			if err != nil {
				a.metrics.Emit(metrics.Error(errors.New("Directive Generation")),
					metrics.With("error", err),
					metrics.With("Level", "Type"),
					metrics.With("Annotaton", annotation.Name),
					metrics.With("Type", typ.Object.Name.Name),
					metrics.With("Params", annotation.Params),
					metrics.With("Arguments", annotation.Arguments),
					metrics.With("Template", annotation.Template))
				return nil, err
			}

			a.metrics.Emit(metrics.Info("Directive Generation: Success"),
				metrics.With("Level", "Type"),
				metrics.With("Directive", len(drs)),
				metrics.With("Annotaton", annotation.Name),
				metrics.With("Type", typ.Object.Name.Name),
				metrics.With("Params", annotation.Params),
				metrics.With("Arguments", annotation.Arguments),
				metrics.With("Template", annotation.Template))

			for _, directive := range drs {
				directives = append(directives, AnnotationWriteDirective{
					WriteDirective: directive,
					Annotation:     annotation.Name,
				})
			}
		}
	}

	return directives, nil
}

// GetPackage returns the annotation generator associated with the giving annotation name.
func (a *AnnotationRegistry) GetPackage(annotation string) (PackageAnnotationGenerator, error) {
	annotation = strings.TrimPrefix(annotation, "@")

	var annon PackageAnnotationGenerator
	var ok bool

	a.ml.RLock()
	{
		annon, ok = a.pkgAnnotations[annotation]
	}
	a.ml.RUnlock()

	if !ok {
		return nil, fmt.Errorf("Package Annotation @%s not found", annotation)
	}

	return annon, nil
}

// MustInterfaceType returns the annotation generator associated with the giving annotation name.
func (a *AnnotationRegistry) MustInterfaceType(annotation string) InterfaceAnnotationGenerator {
	annon, err := a.GetInterfaceType(annotation)
	if err == nil {
		return annon
	}

	panic(err)
}

// MustFunctionType returns the annotation generator associated with the giving annotation name.
func (a *AnnotationRegistry) MustFunctionType(annotation string) FunctionAnnotationGenerator {
	annon, err := a.GetFunctionType(annotation)
	if err == nil {
		return annon
	}

	panic(err)
}

// GetFunctionType returns the annotation generator associated with the giving annotation name.
func (a *AnnotationRegistry) GetFunctionType(annotation string) (FunctionAnnotationGenerator, error) {
	annotation = strings.TrimPrefix(annotation, "@")
	var annon FunctionAnnotationGenerator
	var ok bool

	a.ml.RLock()
	{
		annon, ok = a.functionAnnotations[annotation]
	}
	a.ml.RUnlock()

	if !ok {
		return nil, fmt.Errorf("Function/Method Annotation @%s not found", annotation)
	}

	return annon, nil
}

// GetInterfaceType returns the annotation generator associated with the giving annotation name.
func (a *AnnotationRegistry) GetInterfaceType(annotation string) (InterfaceAnnotationGenerator, error) {
	annotation = strings.TrimPrefix(annotation, "@")
	var annon InterfaceAnnotationGenerator
	var ok bool

	a.ml.RLock()
	{
		annon, ok = a.interfaceAnnotations[annotation]
	}
	a.ml.RUnlock()

	if !ok {
		return nil, fmt.Errorf("Interface Annotation @%s not found", annotation)
	}

	return annon, nil
}

// MustStructType returns the annotation generator associated with the giving annotation name.
func (a *AnnotationRegistry) MustStructType(annotation string) StructAnnotationGenerator {
	annon, err := a.GetStructType(annotation)
	if err == nil {
		return annon
	}

	panic(err)
}

// GetStructType returns the annotation generator associated with the giving annotation name.
func (a *AnnotationRegistry) GetStructType(annotation string) (StructAnnotationGenerator, error) {
	annotation = strings.TrimPrefix(annotation, "@")
	var annon StructAnnotationGenerator
	var ok bool

	a.ml.RLock()
	{
		annon, ok = a.structAnnotations[annotation]
	}
	a.ml.RUnlock()

	if !ok {
		return nil, fmt.Errorf("Struct Annotation @%s not found", annotation)
	}

	return annon, nil
}

// MustType returns the annotation generator associated with the giving annotation name.
func (a *AnnotationRegistry) MustType(annotation string) TypeAnnotationGenerator {
	annon, err := a.GetType(annotation)
	if err == nil {
		return annon
	}

	panic(err)
}

// GetType returns the annotation generator associated with the giving annotation name.
func (a *AnnotationRegistry) GetType(annotation string) (TypeAnnotationGenerator, error) {
	annotation = strings.TrimPrefix(annotation, "@")

	var annon TypeAnnotationGenerator
	var ok bool

	a.ml.RLock()
	{
		annon, ok = a.typeAnnotations[annotation]
	}
	a.ml.RUnlock()

	if !ok {
		return nil, fmt.Errorf("Type Annotation @%s not found", annotation)
	}

	return annon, nil
}

// Register which adds the generator depending on it's type into the appropriate
// registry. It only supports  the following generators:
// 1. TypeAnnotationGenerator (see Package ast#TypeAnnotationGenerator)
// 2. StructAnnotationGenerator (see Package ast#StructAnnotationGenerator)
// 3. InterfaceAnnotationGenerator (see Package ast#InterfaceAnnotationGenerator)
// 4. PackageAnnotationGenerator (see Package ast#PackageAnnotationGenerator)
// Any other type will cause the return of an error.
func (a *AnnotationRegistry) Register(name string, generator interface{}) error {
	switch gen := generator.(type) {
	case PackageAnnotationGenerator:
		a.RegisterPackage(name, gen)
		return nil
	case func(string, AnnotationDeclaration, PackageDeclaration, Package) ([]gen.WriteDirective, error):
		a.RegisterPackage(name, gen)
		return nil
	case TypeAnnotationGenerator:
		a.RegisterType(name, gen)
		return nil
	case func(string, AnnotationDeclaration, TypeDeclaration, PackageDeclaration, Package) ([]gen.WriteDirective, error):
		a.RegisterType(name, gen)
		return nil
	case StructAnnotationGenerator:
		a.RegisterStructType(name, gen)
		return nil
	case func(string, AnnotationDeclaration, StructDeclaration, PackageDeclaration, Package) ([]gen.WriteDirective, error):
		a.RegisterStructType(name, gen)
		return nil
	case InterfaceAnnotationGenerator:
		a.RegisterInterfaceType(name, gen)
		return nil
	case func(string, AnnotationDeclaration, InterfaceDeclaration, PackageDeclaration, Package) ([]gen.WriteDirective, error):
		a.RegisterInterfaceType(name, gen)
		return nil
	default:
		return fmt.Errorf("Generator type for %q not supported: %#v", name, generator)
	}
}

// RegisterInterfaceType adds a interface type level annotation generator into the registry.
func (a *AnnotationRegistry) RegisterInterfaceType(annotation string, generator InterfaceAnnotationGenerator) {
	annotation = strings.TrimPrefix(annotation, "@")
	a.ml.Lock()
	{
		a.interfaceAnnotations[annotation] = generator
	}
	a.ml.Unlock()
}

// RegisterStructType adds a struct type level annotation generator into the registry.
func (a *AnnotationRegistry) RegisterStructType(annotation string, generator StructAnnotationGenerator) {
	annotation = strings.TrimPrefix(annotation, "@")
	a.ml.Lock()
	{
		a.structAnnotations[annotation] = generator
	}
	a.ml.Unlock()
}

// RegisterType adds a type(non-struct, non-interface) level annotation generator into the registry.
func (a *AnnotationRegistry) RegisterType(annotation string, generator TypeAnnotationGenerator) {
	annotation = strings.TrimPrefix(annotation, "@")
	a.ml.Lock()
	{
		a.typeAnnotations[annotation] = generator
	}
	a.ml.Unlock()
}

// RegisterPackage adds a package level annotation generator into the registry.
func (a *AnnotationRegistry) RegisterPackage(annotation string, generator PackageAnnotationGenerator) {
	annotation = strings.TrimPrefix(annotation, "@")
	a.ml.Lock()
	{
		a.pkgAnnotations[annotation] = generator
	}
	a.ml.Unlock()
}

//===========================================================================================================
