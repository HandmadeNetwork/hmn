package utils

import (
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
func DirFS(dir string) fs.FS {
	return dirFS(dir)
}

func containsAny(s, chars string) bool {
	for i := 0; i < len(s); i++ {
		for j := 0; j < len(chars); j++ {
			if s[i] == chars[j] {
				return true
			}
		}
	}
	return false
}

type dirFS string

var _ fs.StatFS = dirFS("")
var _ fs.ReadDirFS = dirFS("")

func (dir dirFS) Open(name string) (fs.File, error) {
	if !fs.ValidPath(name) || runtime.GOOS == "windows" && containsAny(name, `\:`) {
		return nil, &os.PathError{Op: "open", Path: name, Err: os.ErrInvalid}
	}
	f, err := os.Open(string(dir) + "/" + name)
	if err != nil {
		return nil, err // nil fs.File
	}
	return f, nil
}

func (dir dirFS) Stat(name string) (fs.FileInfo, error) {
	if !fs.ValidPath(name) || runtime.GOOS == "windows" && containsAny(name, `\:`) {
		return nil, &os.PathError{Op: "stat", Path: name, Err: os.ErrInvalid}
	}
	f, err := os.Stat(string(dir) + "/" + name)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func (dir dirFS) ReadDir(name string) ([]fs.DirEntry, error) {
	if !fs.ValidPath(name) || runtime.GOOS == "windows" && containsAny(name, `\:`) {
		return nil, &os.PathError{Op: "open", Path: name, Err: os.ErrInvalid}
	}
	d, err := os.ReadDir(string(dir) + "/" + name)
	if err != nil {
		return nil, err
	}
	return d, nil
}
