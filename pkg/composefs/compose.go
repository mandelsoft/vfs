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

package composefs

import (
	"fmt"
	"strings"

	"github.com/mandelsoft/vfs/pkg/utils"
	"github.com/mandelsoft/vfs/pkg/vfs"
)

type ComposedFileSystem struct {
	*utils.MappedFileSystem
	mounts map[string]vfs.FileSystem
}

type adapter struct {
	fs *ComposedFileSystem
}

func (a *adapter) MapPath(path string) (vfs.FileSystem, string) {
	var mountp string
	var mountfs vfs.FileSystem

	for p, fs := range a.fs.mounts {
		if p == path {
			return fs, vfs.PathSeparatorString
		}

		if strings.HasPrefix(path, p+vfs.PathSeparatorString) {
			if len(mountp) < len(p) {
				mountp = p
				mountfs = fs
			}
		}
	}
	if mountfs == nil {
		return a.fs.Base(), path
	}
	return mountfs, path[len(mountp):]
}

func New(root vfs.FileSystem) *ComposedFileSystem {
	fs := &ComposedFileSystem{mounts: map[string]vfs.FileSystem{}}
	fs.MappedFileSystem = utils.NewMappedFileSystem(root, &adapter{fs})
	return fs
}

func (c *ComposedFileSystem) Name() string {
	return fmt.Sprintf("ComposedFileSystem [%s]", c.Base())
}

func (c *ComposedFileSystem) Mount(path string, fs vfs.FileSystem) error {
	mountp, err := vfs.Canonical(c, path, true)
	if err != nil {
		return fmt.Errorf("mount failed: %s", err)
	}
	fi, err := c.Lstat(mountp)
	if err != nil {
		return fmt.Errorf("mount failed: %s", err)
	}
	if !fi.IsDir() {
		return fmt.Errorf("mount failed: mount point %s must be dir", mountp)
	}
	c.mounts[mountp] = fs
	return nil
}
