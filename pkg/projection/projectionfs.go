/*
 * Copyright 2020 Mandelsoft. All rights reserved.
 *  This file is licensed under the Apache Software License, v. 2 except as noted
 *  otherwise in the LICENSE file
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *       http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */

package projection

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mandelsoft/vfs/pkg/utils"
	"github.com/mandelsoft/vfs/pkg/vfs"
)

type ProjectionFileSystem struct {
	*utils.MappedFileSystem
	projection string
}

type adapter struct {
	fs *ProjectionFileSystem
}

func (a *adapter) MapPath(name string) (vfs.FileSystem, string) {
	return a.fs.Base(), vfs.Join(a.fs.Base(), a.fs.projection, name)
}

func New(base vfs.FileSystem, path string) (vfs.FileSystem, error) {
	eff, err := vfs.EvalSymlinks(base, path)
	if err != nil {
		return nil, err
	}
	fs := &ProjectionFileSystem{projection: eff}
	fs.MappedFileSystem = utils.NewMappedFileSystem(base, &adapter{fs})
	return fs, nil
}

func (p *ProjectionFileSystem) Name() string {
	return fmt.Sprintf("ProjectionFilesytem [%s]%s", p.Base().Name(), p.projection)
}

////////////////////////////////////////////////////////////////////////////////////

type oldimpl struct {
	base       vfs.FileSystem
	projection string
}

func NewOld(base vfs.FileSystem, path string) (vfs.FileSystem, error) {
	eff, err := vfs.EvalSymlinks(base, path)
	if err != nil {
		return nil, err
	}
	return &oldimpl{base: base, projection: eff}, nil
}

func (p *oldimpl) Name() string {
	return fmt.Sprintf("ProjectionFilesytem [%s]%s", p.base.Name(), p.projection)
}

func (oldimpl) VolumeName(name string) string {
	return ""
}

func (oldimpl) Normalize(path string) string {
	return path
}

func (oldimpl) Getwd() (string, error) {
	return vfs.PathSeparatorString, nil
}

// isAbs reports whether the path is absolute.
func isAbs(path string) bool {
	return strings.HasPrefix(path, vfs.PathSeparatorString)
}

// on a file outside the base projection it returns the given file name and an error,
// else the given file with the base projection prepended
func (p *oldimpl) pathForBaseFileSystem(path string) (string, error) {

	path = p.base.Normalize(path)
	r := vfs.PathSeparatorString
	links := 0

	for path != "" {
		i := 0
		for i < len(path) && vfs.IsPathSeparator(path[i]) {
			i++
		}
		j := i
		for j < len(path) && !vfs.IsPathSeparator(path[j]) {
			j++
		}

		b := path[i:j]
		path = path[j:]

		switch b {
		case ".", "":
			continue
		case "..":
			r, b = vfs.Split(p.base, r)
			if r == "" {
				r = "/"
			}
			continue
		}
		t := filepath.Join(p.projection, r, b)
		fi, err := p.base.Lstat(t)
		if vfs.Exists_(err) {
			if err != nil && !os.IsPermission(err) {
				return "", err
			}
			if fi.Mode()&os.ModeSymlink != 0 {
				links++
				if links > 255 {
					return "", errors.New("AbsPath: too many links")
				}
				newpath, err := p.base.Readlink(t)
				if err != nil {
					return "", err
				}
				newpath = p.base.Normalize(newpath)
				vol, newpath := vfs.SplitVolume(p.base, newpath)
				if vol != "" {
					return "", fmt.Errorf("volume links not possible: %s: %s", t, vol+newpath)
				}
				if isAbs(newpath) {
					r = "/"
				}
				path = vfs.Join(p.base, newpath, path)
			} else {
				r = vfs.Join(p.base, r, b)
			}
		} else {
			if strings.Contains(path, vfs.PathSeparatorString) {
				return "", err
			}
			r = vfs.Join(p.base, r, b)
		}
	}

	if r == vfs.PathSeparatorString {
		r = p.projection
	} else {
		r = vfs.Join(p.base, p.projection, r)
	}
	return r, nil
}

func (p *oldimpl) Chtimes(name string, atime, mtime time.Time) (err error) {
	if name, err = p.pathForBaseFileSystem(name); err != nil {
		return &os.PathError{Op: "chtimes", Path: name, Err: err}
	}
	return p.base.Chtimes(name, atime, mtime)
}

func (p *oldimpl) Chmod(name string, mode os.FileMode) (err error) {
	if name, err = p.pathForBaseFileSystem(name); err != nil {
		return &os.PathError{Op: "chmod", Path: name, Err: err}
	}
	return p.base.Chmod(name, mode)
}

func (p *oldimpl) Stat(name string) (fi os.FileInfo, err error) {
	if name, err = p.pathForBaseFileSystem(name); err != nil {
		return nil, &os.PathError{Op: "stat", Path: name, Err: err}
	}
	return p.base.Stat(name)
}

func (p *oldimpl) Rename(oldname, newname string) (err error) {
	if oldname, err = p.pathForBaseFileSystem(oldname); err != nil {
		return &os.PathError{Op: "rename", Path: oldname, Err: err}
	}
	if newname, err = p.pathForBaseFileSystem(newname); err != nil {
		return &os.PathError{Op: "rename", Path: newname, Err: err}
	}
	return p.base.Rename(oldname, newname)
}

func (p *oldimpl) RemoveAll(name string) (err error) {
	if name, err = p.pathForBaseFileSystem(name); err != nil {
		return &os.PathError{Op: "remove_all", Path: name, Err: err}
	}
	return p.base.RemoveAll(name)
}

func (p *oldimpl) Remove(name string) (err error) {
	if name, err = p.pathForBaseFileSystem(name); err != nil {
		return &os.PathError{Op: "remove", Path: name, Err: err}
	}
	return p.base.Remove(name)
}

func (p *oldimpl) OpenFile(name string, flag int, mode os.FileMode) (f vfs.File, err error) {
	if name, err = p.pathForBaseFileSystem(name); err != nil {
		return nil, &os.PathError{Op: "openfile", Path: name, Err: err}
	}
	sourcef, err := p.base.OpenFile(name, flag, mode)
	if err != nil {
		return nil, err
	}
	return utils.NewRenamedFile(name, sourcef), nil
}

func (p *oldimpl) Open(name string) (f vfs.File, err error) {
	if name, err = p.pathForBaseFileSystem(name); err != nil {
		return nil, &os.PathError{Op: "open", Path: name, Err: err}
	}
	sourcef, err := p.base.Open(name)
	if err != nil {
		return nil, err
	}
	return utils.NewRenamedFile(name, sourcef), nil
}

func (p *oldimpl) Mkdir(name string, mode os.FileMode) (err error) {
	if name, err = p.pathForBaseFileSystem(name); err != nil {
		return &os.PathError{Op: "mkdir", Path: name, Err: err}
	}
	return p.base.Mkdir(name, mode)
}

func (p *oldimpl) MkdirAll(name string, mode os.FileMode) (err error) {
	if name, err = p.pathForBaseFileSystem(name); err != nil {
		return &os.PathError{Op: "mkdir", Path: name, Err: err}
	}
	return p.base.MkdirAll(name, mode)
}

func (p *oldimpl) Create(name string) (f vfs.File, err error) {
	if name, err = p.pathForBaseFileSystem(name); err != nil {
		return nil, &os.PathError{Op: "create", Path: name, Err: err}
	}
	sourcef, err := p.base.Create(name)
	if err != nil {
		return nil, err
	}
	return utils.NewRenamedFile(name, sourcef), nil
}

func (p *oldimpl) Lstat(name string) (os.FileInfo, error) {
	name, err := p.pathForBaseFileSystem(name)
	if err != nil {
		return nil, &os.PathError{Op: "lstat", Path: name, Err: err}
	}
	return p.base.Lstat(name)
}

func (p *oldimpl) Symlink(oldname, newname string) error {
	oldname, err := p.pathForBaseFileSystem(oldname)
	if err != nil {
		return &os.LinkError{Op: "symlink", Old: oldname, New: newname, Err: err}
	}
	newname, err = p.pathForBaseFileSystem(newname)
	if err != nil {
		return &os.LinkError{Op: "symlink", Old: oldname, New: newname, Err: err}
	}
	return p.base.Symlink(oldname, newname)
}

func (p *oldimpl) Readlink(name string) (string, error) {
	name, err := p.pathForBaseFileSystem(name)
	if err != nil {
		return "", &os.PathError{Op: "readlink", Path: name, Err: err}
	}
	return p.base.Readlink(name)
}
