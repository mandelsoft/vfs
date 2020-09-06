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

package utils

import (
	"github.com/mandelsoft/vfs/pkg/vfs"
)

type FileData interface {
	IsDir() bool
	IsSymlink() bool
	Get(name string) (FileData, error)
	GetSymlink() string
}

func EvaluatePath(fs vfs.FileSystem, root FileData, name string, link ...bool) (FileData, string, FileData, string, error) {
	var data []FileData
	var path string
	var dir bool

	_, elems, _ := vfs.SplitPath(fs, name)
	getlink := true
	if len(link) > 0 {
		getlink = link[0]
	}
outer:
	for {
		path = "/"
		data = []FileData{root}
		dir = true

		for i := 0; i < len(elems); i++ {
			e := elems[i]
			cur := len(data) - 1
			switch e {
			case ".":
				if !dir {
					return nil, "", nil, "", vfs.ErrNotDir
				}
				continue
			case "..":
				if !dir {
					return nil, "", nil, "", vfs.ErrNotDir
				}
				if len(data) > 1 {
					data = data[:cur]
					path, _ = vfs.Split(fs, path)
				}
				continue
			}
			next, err := data[cur].Get(e)
			if vfs.IsErrNotDir(err) {
				return nil, "", nil, "", vfs.NewPathError("", path, err)
			}
			if vfs.IsErrNotExist(err) {
				if i == len(elems)-1 {
					return data[cur], path, nil, e, nil
				}
				return nil, "", nil, "", vfs.NewPathError("", vfs.Join(fs, path, e), err)
			}
			if !next.IsSymlink() || (!getlink && i == len(elems)-1) {
				dir = next.IsDir()
				path = vfs.Join(fs, path, e)
				data = append(data, next)
				continue
			}
			link := next.GetSymlink()
			_, nested, rooted := vfs.SplitPath(fs, link)
			if rooted {
				elems = append(nested, elems[i+1:]...)
				i = 0
				continue outer
			}
			elems = append(elems[:i], append(nested, elems[i+1:]...)...)
			i--
		}
		break
	}
	if path == vfs.PathSeparatorString {
		return root, path, root, "", nil
	}
	d, b := vfs.Split(fs, path)
	if d == "" {
		return root, vfs.PathSeparatorString, data[len(data)-1], b, nil
	}
	return data[len(data)-2], d, data[len(data)-1], b, nil
}
