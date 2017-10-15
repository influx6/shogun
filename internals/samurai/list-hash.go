package samurai

import (
	"os"

	"github.com/influx6/faux/metrics"
	"github.com/influx6/faux/vfiles"
	"github.com/influx6/gobuild/build"
	"github.com/influx6/gobuild/srcpath"
	"github.com/influx6/moz/ast"
)

// PackageHashList holds a list of hashes from a main package and
// all other subpackages retrieved.
type PackageHashList struct {
	Dir       string              `json:"dir"`
	SuperHash string              `json:"super_hash"`
	Main      HashList            `json:"main"`
	Subs      map[string]HashList `json:"subs"`
}

// ListPackageHash returns all functions retrieved from the directory filtered by the build.Context.
func ListPackageHash(vlog, events metrics.Metrics, targetDir string, ctx build.Context) (PackageHashList, error) {
	var list PackageHashList
	list.Dir = targetDir
	list.Subs = make(map[string]HashList)

	// Build shogunate directory itself first.
	var err error
	list.Main, err = HashPackages(vlog, events, targetDir, ctx)
	if err != nil {
		events.Emit(metrics.Errorf("Failed to generate function list : %+q", err))
		return list, err
	}

	var hash []byte
	hash = append(hash, []byte(list.Main.Hash)...)

	if err = vfiles.WalkDirSurface(targetDir, func(rel string, abs string, info os.FileInfo) error {
		if !info.IsDir() {
			return nil
		}

		res, err2 := HashPackages(vlog, events, abs, ctx)
		if err2 != nil {
			if err2 == ErrSkipDir {
				return nil
			}

			return err2
		}

		res.RelPath = rel
		list.Subs[res.Path] = res
		hash = append(hash, []byte(res.Hash)...)
		return nil
	}); err != nil {
		events.Emit(metrics.Error(err).With("dir", targetDir))
		return list, err
	}

	list.SuperHash = string(hash)
	return list, nil
}

// HashList holds the list of processed functions from individual packages.
type HashList struct {
	Path     string            `json:"path"`
	RelPath  string            `json:"relpath,omitempty"`
	Hash     string            `json:"hash"`
	Package  string            `json:"package"`
	Packages map[string]string `json:"packages"`
}

// HashPackages iterates all directories and generates package hashes of all declared functions
// matching the shegun format.
func HashPackages(vlog, events metrics.Metrics, dir string, ctx build.Context) (HashList, error) {
	var pkgFuncs HashList
	pkgFuncs.Path = dir
	pkgFuncs.Packages = make(map[string]string)
	pkgFuncs.Package, _ = srcpath.RelativeToSrc(dir)

	pkgs, err := ast.FilteredPackageWithBuildCtx(vlog, dir, ctx)
	if err != nil {
		if _, ok := err.(*build.NoGoError); ok {
			return pkgFuncs, ErrSkipDir
		}

		events.Emit(metrics.Error(err).With("dir", dir))
		return pkgFuncs, err
	}

	if len(pkgs) == 0 {
		return pkgFuncs, ErrSkipDir
	}

	pkgItem := pkgs[0]
	if pkgItem.HasAnnotation("@shogunIgnore") {
		return pkgFuncs, ErrSkipDir
	}

	pkgHash, err := generateHash(pkgItem.Files)
	if err != nil {
		return pkgFuncs, err
	}

	pkgFuncs.Packages[pkgItem.Path] = pkgHash

	pkgFuncs.Hash = string(pkgHash)

	return pkgFuncs, nil
}
