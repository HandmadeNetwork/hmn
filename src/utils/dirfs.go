package utils

import (
	"errors"
	"io/fs"
	"os"
	"runtime"
)

// DirFS returns a file system (an fs.FS) for the tree of files rooted at the directory dir.
//
// Note that DirFS("/prefix") only guarantees that the Open calls it makes to the
// operating system will begin with "/prefix": DirFS("/prefix").Open("file") is the
// same as os.Open("/prefix/file"). So if /prefix/file is a symbolic link pointing outside
// the /prefix tree, then using DirFS does not stop the access any more than using
// os.Open does. Additionally, the root of the fs.FS returned for a relative path,
// DirFS("prefix"), will be affected by later calls to Chdir. DirFS is therefore not
// a general substitute for a chroot-style security mechanism when the directory tree
// contains arbitrary content.
//
// The result implements fs.StatFS AND fs.ReadDirFS because god dammit why not.
//
// Implementation copy-pasted from Go 1.20.2.
func DirFS(dir string) fs.FS {
	return dirFS(dir)
}

type dirFS string

var _ fs.StatFS = dirFS("")
var _ fs.ReadDirFS = dirFS("")

func (dir dirFS) Open(name string) (fs.File, error) {
	fullname, err := dir.join(name)
	if err != nil {
		return nil, &os.PathError{Op: "stat", Path: name, Err: err}
	}
	f, err := os.Open(fullname)
	if err != nil {
		// DirFS takes a string appropriate for GOOS,
		// while the name argument here is always slash separated.
		// dir.join will have mixed the two; undo that for
		// error reporting.
		err.(*os.PathError).Path = name
		return nil, err
	}
	return f, nil
}

func (dir dirFS) Stat(name string) (fs.FileInfo, error) {
	fullname, err := dir.join(name)
	if err != nil {
		return nil, &os.PathError{Op: "stat", Path: name, Err: err}
	}
	f, err := os.Stat(fullname)
	if err != nil {
		// See comment in dirFS.Open.
		err.(*os.PathError).Path = name
		return nil, err
	}
	return f, nil
}

func (dir dirFS) ReadDir(name string) ([]fs.DirEntry, error) {
	fullname, err := dir.join(name)
	if err != nil {
		return nil, &os.PathError{Op: "stat", Path: name, Err: err}
	}
	d, err := os.ReadDir(fullname)
	if err != nil {
		// See comment in dirFS.Open.
		err.(*os.PathError).Path = name
		return nil, err
	}
	return d, nil
}

func fromFS(path string) (string, error) {
	if runtime.GOOS == "plan9" {
		if len(path) > 0 && path[0] == '#' {
			return "", os.ErrInvalid
		}
	}
	for i := range path {
		if path[i] == 0 {
			return "", os.ErrInvalid
		}
	}
	return path, nil
}

// join returns the path for name in dir.
func (dir dirFS) join(name string) (string, error) {
	if dir == "" {
		return "", errors.New("os: DirFS with empty root")
	}
	if !fs.ValidPath(name) {
		return "", os.ErrInvalid
	}
	name, err := fromFS(name)
	if err != nil {
		return "", os.ErrInvalid
	}
	if os.IsPathSeparator(dir[len(dir)-1]) {
		return string(dir) + name, nil
	}
	return string(dir) + string(os.PathSeparator) + name, nil
}
