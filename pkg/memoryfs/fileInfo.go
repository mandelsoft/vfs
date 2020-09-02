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
	"time"
)

///////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// fileInfo implementing os.FileInfo
///////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type fileInfo struct {
	*fileData
	name string
}

func newFileInfo(name string, file *fileData) os.FileInfo {
	return &fileInfo{name: name, fileData: file}
}

var _ os.FileInfo = &fileInfo{}

func (f *fileInfo) Name() string {
	f.Lock()
	defer f.Unlock()
	return f.name
}

func (f *fileInfo) Mode() os.FileMode {
	f.Lock()
	defer f.Unlock()
	return f.mode
}

func (f *fileInfo) ModTime() time.Time {
	f.Lock()
	defer f.Unlock()
	return f.modtime
}

func (f *fileInfo) IsDir() bool {
	f.Lock()
	defer f.Unlock()
	return f.fileData.IsDir()
}

func (f *fileInfo) Sys() interface{} { return nil }

func (f *fileInfo) Size() int64 {
	f.Lock()
	defer f.Unlock()
	if f.fileData.IsDir() {
		return int64(42)
	}
	return int64(len(f.data))
}
