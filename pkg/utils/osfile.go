/*
 * Copyright 2024 Mandelsoft. All rights reserved.
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

// BackingOSFile is an optional interface for a file
// object providing access to an os.File used to back
// the file.
type BackingOSFile interface {
	OSFile() *os.File
}

// OSFile return the os.File used to back
// the given virtual file. It returns nil,
// if is not backed by an os.File.
func OSFile(f vfs.File) *os.File {
	if o, ok := f.(BackingOSFile); ok {
		return o.OSFile()
	}
	return nil
}
