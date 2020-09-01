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

package cwd

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/mandelsoft/vfs/pkg/vfs"
)

type CWD struct {
	base vfs.FileSystem
	vol  string
	cwd  string
}

func New(base vfs.FileSystem, path string) (vfs.FileSystemWithWorkingDirectory, error) {
	real, err := vfs.Canonical(base, path, true)
	if err != nil {
		return nil, err
	}
	dir, err := base.Stat(real)
	if err != nil {
		return nil, err
	}
	if !dir.IsDir() {
		return nil, &os.PathError{Op: "readdir", Path: path, Err: errors.New("not a dir")}
	}
	if old, ok := base.(*CWD); ok {
		base = old
	}
	return &CWD{base, base.VolumeName(real), real}, nil
}

func (c *CWD) Chdir(path string) error {
	real, err := vfs.Canonical(c, path, true)
	if err != nil {
		return err
	}
	fi, err := c.Lstat(real)
	if err != nil {
		return err
	}
	if !fi.IsDir() {
		return &os.PathError{Op: "chdir", Path: path, Err: errors.New("no dir")}
	}
	c.cwd = real
	return nil
}

func (c *CWD) Name() string {
	return fmt.Sprintf("%s(%s)", c.base.Name(), c.cwd)
}

func (c *CWD) VolumeName(name string) string {
	return c.base.VolumeName(name)
}

func (c *CWD) Normalize(path string) string {
	return c.base.Normalize(path)
}

func (c *CWD) Getwd() (string, error) {
	return c.cwd, nil
}

func (c *CWD) realPath(path string) (string, error) {
	vol, path := vfs.SplitVolume(c.base, path)

	if vol != c.vol {
		return "", fmt.Errorf("volume mismatch")
	}
	if vfs.IsAbs(c, path) {
		return vol + path, nil
	}
	return vfs.Join(c.base, c.cwd, path), nil
}

func (c *CWD) Create(name string) (vfs.File, error) {
	abs, err := c.realPath(name)
	if err != nil {
		return nil, err
	}
	return c.base.Create(abs)
}

func (c *CWD) Mkdir(name string, perm os.FileMode) error {
	abs, err := c.realPath(name)
	if err != nil {
		return err
	}
	return c.base.Mkdir(abs, perm)
}

func (c *CWD) MkdirAll(path string, perm os.FileMode) error {
	abs, err := c.realPath(path)
	if err != nil {
		return err
	}
	return c.base.MkdirAll(abs, perm)
}

func (c *CWD) Open(name string) (vfs.File, error) {
	abs, err := c.realPath(name)
	if err != nil {
		return nil, err
	}
	return c.base.Open(abs)
}

func (c *CWD) OpenFile(name string, flag int, perm os.FileMode) (vfs.File, error) {
	abs, err := c.realPath(name)
	if err != nil {
		return nil, err
	}
	return c.base.OpenFile(abs, flag, perm)
}

func (c *CWD) Remove(name string) error {
	abs, err := c.realPath(name)
	if err != nil {
		return err
	}
	return c.base.Remove(abs)
}

func (c *CWD) RemoveAll(path string) error {
	abs, err := c.realPath(path)
	if err != nil {
		return err
	}
	return c.base.RemoveAll(abs)
}

func (c *CWD) Rename(oldname, newname string) error {
	absnew, err := c.realPath(newname)
	if err != nil {
		return err
	}
	absold, err := c.realPath(oldname)
	if err != nil {
		return err
	}
	return c.base.Rename(absold, absnew)
}

func (c *CWD) Stat(name string) (os.FileInfo, error) {
	abs, err := c.realPath(name)
	if err != nil {
		return nil, err
	}
	return c.base.Stat(abs)
}

func (c *CWD) Chmod(name string, mode os.FileMode) error {
	abs, err := c.realPath(name)
	if err != nil {
		return err
	}
	return c.base.Chmod(abs, mode)
}

func (c *CWD) Chtimes(name string, atime time.Time, mtime time.Time) error {
	abs, err := c.realPath(name)
	if err != nil {
		return err
	}
	return c.base.Chtimes(abs, atime, mtime)
}

func (c *CWD) Lstat(name string) (os.FileInfo, error) {
	abs, err := c.realPath(name)
	if err != nil {
		return nil, err
	}
	return c.base.Lstat(abs)
}

func (c *CWD) Symlink(oldname, newname string) error {
	abs, err := c.realPath(newname)
	if err != nil {
		return err
	}
	return c.base.Symlink(oldname, abs)
}

func (c *CWD) Readlink(name string) (string, error) {
	abs, err := c.realPath(name)
	if err != nil {
		return "", err
	}
	return c.base.Readlink(abs)
}
