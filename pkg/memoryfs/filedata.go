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

package memoryfs

import (
	"os"
	"sync"
	"time"

	"github.com/mandelsoft/vfs/pkg/utils"
	"github.com/mandelsoft/vfs/pkg/vfs"
)

type fileData struct {
	sync.Mutex
	data    []byte
	entries DirectoryEntries
	mode    os.FileMode
	modtime time.Time
}

func asFileData(data utils.FileData) *fileData {
	if data == nil {
		return nil
	}
	return data.(*fileData)
}

func (f *fileData) IsDir() bool {
	return f.mode&os.ModeType == os.ModeDir
}

func (f *fileData) IsFile() bool {
	return f.mode&os.ModeType == 0
}

func (f *fileData) IsSymlink() bool {
	return (f.mode & os.ModeType) == os.ModeSymlink
}

func createFile(perm os.FileMode) *fileData {
	return &fileData{mode: os.ModeTemporary | (perm & os.ModePerm), modtime: time.Now()}
}

func createDir(perm os.FileMode) *fileData {
	return &fileData{mode: os.ModeDir | os.ModeTemporary | (perm & os.ModePerm), entries: DirectoryEntries{}, modtime: time.Now()}
}

func createSymlink(link string, perm os.FileMode) *fileData {
	return &fileData{mode: os.ModeSymlink | os.ModeTemporary | (perm & os.ModePerm), data: []byte(link), modtime: time.Now()}
}

func (f *fileData) GetSymlink() string {
	f.Lock()
	defer f.Unlock()
	if f.IsSymlink() {
		return string(f.data)
	}
	return ""
}
func (f *fileData) SetMode(mode os.FileMode) {
	f.Lock()
	f.mode = mode
	f.Unlock()
}

func (f *fileData) Chmod(mode os.FileMode) {
	f.Lock()
	f.mode = (f.mode & (^os.ModePerm)) | (mode & os.ModePerm)
	f.Unlock()
}

func (f *fileData) SetModTime(mtime time.Time) {
	f.Lock()
	f.setModTime(mtime)
	f.Unlock()
}

func (f *fileData) setModTime(mtime time.Time) {
	f.modtime = mtime
}

func (f *fileData) AddH(name string, s *fileData) (vfs.File, error) {
	err := f.Add(name, s)
	if err != nil {
		return nil, err
	}
	return newFileHandle(name, s), nil
}

func (f *fileData) Add(name string, s *fileData) error {
	f.Lock()
	defer f.Unlock()
	if !f.IsDir() {
		return ErrNotDir
	}
	if _, ok := f.entries[name]; ok {
		return os.ErrExist
	}
	f.entries.Add(name, s)
	f.setModTime(time.Now())
	return nil
}

func (f *fileData) Get(name string) (utils.FileData, error) {
	f.Lock()
	defer f.Unlock()
	if !f.IsDir() {
		return nil, ErrNotDir
	}
	e, ok := f.entries[name]
	if ok {
		return e, nil
	}
	return nil, os.ErrNotExist
}

func (f *fileData) Del(name string) error {
	f.Lock()
	defer f.Unlock()
	if !f.IsDir() {
		return ErrNotDir
	}
	_, ok := f.entries[name]
	if !ok {
		return os.ErrNotExist
	}
	delete(f.entries, name)
	return nil
}
