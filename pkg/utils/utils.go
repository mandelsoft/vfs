/*
 * Copyright 2022 Mandelsoft. All rights reserved.
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
	"os"

	"github.com/mandelsoft/vfs/pkg/vfs"
)

type RenamedFile struct {
	vfs.File
	name string
}

var _ BackingOSFile = (*RenamedFile)(nil)

func NewRenamedFile(name string, file vfs.File) vfs.File {
	return &RenamedFile{file, name}
}

func (r *RenamedFile) Name() string {
	return r.name
}

func (r *RenamedFile) OSFile() *os.File {
	return OSFile(r.File)
}
