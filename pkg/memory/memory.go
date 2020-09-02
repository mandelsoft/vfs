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

package memory

import (
	"errors"
	"os"
	"strings"
	"time"

	"github.com/mandelsoft/vfs/pkg/vfs"
)

type MemoryFileSystem struct {
	root *fileData
}

func New() vfs.FileSystem {
	return &MemoryFileSystem{createDir(os.ModePerm)}
}

func (MemoryFileSystem) Name() string {
	return "MemoryFileSystem"
}

func (MemoryFileSystem) VolumeName(name string) string {
	return ""
}

func (MemoryFileSystem) Normalize(path string) string {
	return path
}

func (*MemoryFileSystem) Getwd() (string, error) {
	return vfs.PathSeparatorString, nil
}

func (m *MemoryFileSystem) findFile(name string, link ...bool) (*fileData, string, error) {
	_, _, f, n, err := m.createInfo(name, link...)
	if err != nil {
		return nil, n, err
	}
	if f == nil {
		err = os.ErrNotExist
	}
	return f, n, err
}

func (m *MemoryFileSystem) createInfo(name string, link ...bool) (*fileData, string, *fileData, string, error) {
	var data []*fileData
	var path string

	_, elems, _ := vfs.SplitPath(m, name)
	getlink := true
	if len(link) > 0 {
		getlink = link[0]
	}
outer:
	for {
		path = "/"
		data = []*fileData{m.root}

		for i := 0; i < len(elems); i++ {
			e := elems[i]
			cur := len(data) - 1
			switch e {
			case ".":
				continue
			case "..":
				if len(data) > 1 {
					data = data[:cur]
					path, _ = vfs.Split(m, path)
				}
				continue
			}
			next, err := data[cur].Get(e)
			switch err {
			case ErrNoDir:
				return nil, "", nil, "", &os.PathError{Op: "", Path: path, Err: err}
			case os.ErrNotExist:
				if i == len(elems)-1 {
					return data[cur], path, nil, e, nil
				}
				return nil, "", nil, "", &os.PathError{Op: "", Path: vfs.Join(m, path, e), Err: err}
			}
			if !next.IsSymlink() || (!getlink && i == len(elems)-1) {
				path = vfs.Join(m, path, e)
				data = append(data, next)
				continue
			}
			l := next.GetSymlink()
			_, nested, rooted := vfs.SplitPath(m, l)
			if rooted {
				elems = append(nested, elems[i+1:]...)
				i = 0
				continue outer
			}
			elems = append(append(elems[:i], nested...), elems[i+1:]...)
			i--
		}
		break
	}
	if path == vfs.PathSeparatorString {
		return m.root, path, m.root, "", nil
	}
	d, b := vfs.Split(m, path)
	if d == "" {
		return m.root, vfs.PathSeparatorString, data[len(data)-1], b, nil
	}
	return data[len(data)-2], d, data[len(data)-1], b, nil
}

func (m *MemoryFileSystem) Create(name string) (vfs.File, error) {
	parent, _, f, n, err := m.createInfo(name)
	if err != nil {
		return nil, err
	}
	if f != nil {
		return nil, os.ErrExist
	}
	return parent.AddH(n, createFile(os.ModePerm))
}

func (m *MemoryFileSystem) Mkdir(name string, perm os.FileMode) error {
	parent, _, f, n, err := m.createInfo(name)
	if err != nil {
		return err
	}
	if f != nil {
		return os.ErrExist
	}
	return parent.Add(n, createDir(perm))
}

func (m *MemoryFileSystem) MkdirAll(path string, perm os.FileMode) error {
	path, err := vfs.Canonical(m, path, false)
	if err != nil {
		return err
	}
	_, elems, _ := vfs.SplitPath(m, path)
	parent := m.root
	for i, e := range elems {
		next, err := parent.Get(e)
		if err != nil && err != os.ErrNotExist {
			return &os.PathError{Op: "mkdirall", Path: strings.Join(elems[:i+1], vfs.PathSeparatorString), Err: err}
		}
		if next == nil {
			next = createDir(perm)
			parent.Add(e, next)
		}
		parent = next
	}
	return nil
}

func (m *MemoryFileSystem) Open(name string) (vfs.File, error) {
	f, n, err := m.findFile(name)
	if err != nil {
		return nil, err
	}
	return newFileHandle(n, f), nil
}

func (m *MemoryFileSystem) OpenFile(name string, flags int, perm os.FileMode) (vfs.File, error) {
	dir, _, f, n, err := m.createInfo(name)
	if err != nil {
		return nil, err
	}
	if f == nil {
		if flags&(os.O_CREATE) == 0 {
			return nil, &os.PathError{Op: "create", Path: name, Err: os.ErrNotExist}
		}
		f := createFile(perm)
		err = dir.Add(n, f)
		if err != nil {
			return nil, &os.PathError{Op: "create", Path: name, Err: err}
		}
	}
	h := newFileHandle(n, f)

	if flags == os.O_RDONLY {
		h.readOnly = true
	} else {
		if flags&os.O_APPEND != 0 {
			_, err = h.Seek(0, os.SEEK_END)
		}
		if err == nil && flags&os.O_TRUNC > 0 && flags&(os.O_RDWR|os.O_WRONLY) > 0 {
			err = h.Truncate(0)
		}
		if err != nil {
			h.Close()
			return nil, err
		}
	}
	return h, nil
}

func (m *MemoryFileSystem) Remove(name string) error {
	dir, _, f, n, err := m.createInfo(name)
	if err != nil {
		return err
	}

	if f == nil {
		return os.ErrNotExist
	}
	f.Lock()
	defer f.Unlock()
	if f.IsDir() {
		if len(f.entries) > 0 {
			return &os.PathError{Op: "remove", Path: name, Err: ErrNotEmpty}
		}
	}
	if n == "" {
		return errors.New("cannot delete root dir")
	}
	return dir.Del(n)
}

func (m *MemoryFileSystem) RemoveAll(name string) error {
	dir, _, _, n, err := m.createInfo(name)
	if err != nil {
		return err
	}
	if n == "" {
		return errors.New("cannot delete root dir")
	}
	return dir.Del(n)
}

func (m *MemoryFileSystem) Rename(oldname, newname string) error {
	odir, _, fo, o, err := m.createInfo(oldname)
	if err != nil {
		return err
	}
	if o == "" {
		return errors.New("cannot rename root dir")
	}
	ndir, _, fn, n, err := m.createInfo(newname)
	if err != nil {
		return err
	}
	if fo == nil {
		return os.ErrNotExist
	}
	if fn != nil {
		return os.ErrExist
	}

	err = ndir.Add(n, fo)
	if err == nil {
		odir.Del(o)
	}
	return err
}

func (m *MemoryFileSystem) Lstat(name string) (os.FileInfo, error) {
	f, n, err := m.findFile(name, false)
	if err != nil {
		return nil, err
	}
	return newFileInfo(n, f), nil
}

func (m *MemoryFileSystem) Stat(name string) (os.FileInfo, error) {
	f, n, err := m.findFile(name)
	if err != nil {
		return nil, err
	}
	if f == nil {
		return nil, os.ErrNotExist
	}
	return newFileInfo(n, f), nil
}

func (m *MemoryFileSystem) Chmod(name string, mode os.FileMode) error {
	f, _, err := m.findFile(name)
	if err != nil {
		return err
	}
	f.Chmod(mode)
	return nil
}

func (m *MemoryFileSystem) Chtimes(name string, atime time.Time, mtime time.Time) error {
	f, _, err := m.findFile(name)
	if err != nil {
		return err
	}
	f.SetModTime(mtime)
	return nil
}

func (m *MemoryFileSystem) Symlink(oldname, newname string) error {
	parent, _, _, n, err := m.createInfo(newname)
	if err != nil {
		return err
	}
	return parent.Add(n, createSymlink(oldname, os.ModePerm))
}

func (m *MemoryFileSystem) Readlink(name string) (string, error) {
	f, _, err := m.findFile(name, false)
	if err != nil {
		return "", err
	}
	if f.IsSymlink() {
		return f.GetSymlink(), nil
	}
	return "", &os.PathError{Op: "readlink", Path: name, Err: errors.New("no symlink")}
}
