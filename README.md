
## VFS - A Virtual Filesystem for GO


[![CI Build status](https://app.travis-ci.com/mandelsoft/vfs.svg?branch=master)](https://app.travis-ci.com/github/mandelsoft/vfs)
[![Go Report Card](https://goreportcard.com/badge/github.com/mandelsoft/vfs)](https://goreportcard.com/report/github.com/mandelsoft/vfs)


A virtual filesystem enables programs to transparently work on any kind
of filesystem-like data source besides the real operating system filesystem
using a uniform single API.

This project provides an API that can be used instead of the native 
OS filesytem API. Below this API there might be several technical
implementations that simulate a uniform filesystem for all its users.

- package `osfs` provides access to the real operating system filesystem
  observing the current working directory (see [godoc](https://pkg.go.dev/github.com/mandelsoft/vfs/pkg/osfs)).
- package `memoryfs` provides a pure memory based file system supporting
  files, directories and symbolic links (see [godoc](https://pkg.go.dev/github.com/mandelsoft/vfs/pkg/memoryfs)).
- package `composefs` provides a virtual filesystem composable of
  multiple other virtual filesystems, that can be mounted on top of
  a root file system (see [godoc](https://pkg.go.dev/github.com/mandelsoft/vfs/pkg/composefs)).
- package `layerfs` provides a filesystem layer on top of a base filesystem.
  The layer can be implemented by any other virtual filesystem, for example
  a memory filesystem (see [godoc](https://pkg.go.dev/github.com/mandelsoft/vfs/pkg/layerfs)).
- package `yamlfs` provides a filesystem based on the structure and content of a
  yaml document (see [godoc](https://pkg.go.dev/github.com/mandelsoft/vfs/pkg/yamlfs)).
  The document can even be changed by filesystem operations.
  
Besides those new implementations for a virtual filesystem there are 
some implementation modifying the bahaviour of a base filesystem:

- package `readonlyfs` provides a read-only view of a base filesystem (see [godoc](https://pkg.go.dev/github.com/mandelsoft/vfs/pkg/readonlyfs)).
- package `cwdfs` provides the notion of a current working directory for
  any base filesystem (see [godoc](https://pkg.go.dev/github.com/mandelsoft/vfs/pkg/cwdfs)).
- package `projectionfs` provides a filesystem based on a dedicated directory
  of a base filesystem (see [godoc](https://pkg.go.dev/github.com/mandelsoft/vfs/pkg/projectionfs)).
  
All the implementation packages provide some `New` function to create an
instance of the dedicated filesystem type.

To work with the OS filesystem just create an instance of
the `osfs`:

```golang
  import "github.com/mandelsoft/vfs/pkg/osfs"

  ...

  fs := osfs.New()
```

Now the standard go filesystem related `os` API can be used just by replacing
the package `os` by the instance of the virtual filesystem.

```golang

  f, err := fs.Open()
  if err!=nil {
    return nil, err
  }
  defer f.Close()
  return ioutil.ReadAll(f)
```

To support this the package `vfs` provides a common interface `FileSystem` that 
offers methods similar to the `os` file operations. Additionally an own
`File` interface is provided that replaces the struct `os.File` for the use
in the context of the virtual filesystem. (see [godoc](https://pkg.go.dev/github.com/mandelsoft/vfs/pkg/vfs))

A `FileSystem` may offer a temp directory and a current working directory.
The typical implementations for new kinds of a filesystem do not provide
these features, they rely on the orchestration with dedicated implementations,
for example a `cwdfs.WorkingDirectoryFileSystem` or a
`composedfs.ComposedFileSystem`, which allows mounting a temporary filesystem.
The package `osfs` supports creating a temporary os filesystem based
virtual filesystem residing in a temporary operating system directory.

Additionally, the interface `VFS` includes the standard filesystem operations
and some implementation independent utility functions based on a virtual
filesystem known from the `os`, `ìoutil` and `filepath` packages.
The function `vfs.New(fs)` can be used to create such a wrapper for
any virtual filesystem.

A virtual filesystem can be used as `io/fs.FS` or `io/fs.ReadDirFS`.
Because of the Go typesystem and the stripped interface `io/fs.File`,
this is not directly possible. But any virtual filesystem can be converted
by a type converting wrapper function `vfs.AsIoFS(fs)`.