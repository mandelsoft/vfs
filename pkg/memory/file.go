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
	"bytes"
	"errors"
	"io"
	"os"
	"time"

	"github.com/mandelsoft/vfs/pkg/vfs"
)

type file struct {
	// atomic requires 64-bit alignment for struct field access
	offset       int64
	readDirCount int64
	closed       bool
	readOnly     bool
	fileData     *fileData
	name         string
}

var _ vfs.File = &file{}

func newFileHandle(name string, data *fileData) *file {
	return &file{name: name, fileData: data}
}

func (f file) Data() *fileData {
	return f.fileData
}

func (f *file) Open() error {
	f.fileData.Lock()
	f.offset = 0
	f.readDirCount = 0
	f.closed = false
	f.fileData.Unlock()
	return nil
}

func (f *file) Close() error {
	f.fileData.Lock()
	f.closed = true
	f.fileData.Unlock()
	return nil
}

func (f *file) Name() string {
	return f.name
}

func (f *file) Stat() (os.FileInfo, error) {
	return newFileInfo(f.name, f.fileData), nil
}

func (f *file) Sync() error {
	return nil
}

func (f *file) Readdir(count int) (files []os.FileInfo, err error) {
	if !f.fileData.IsDir() {
		return nil, &os.PathError{Op: "readdir", Path: f.name, Err: ErrNoDir}
	}
	var outLength int64

	f.fileData.Lock()
	defer f.fileData.Unlock()

	files = f.fileData.entries.Files()
	if f.readDirCount >= int64(len(files)) {
		return nil, io.EOF
	}
	files = files[f.readDirCount:]
	if count > 0 {
		if len(files) < count {
			outLength = int64(len(files))
		} else {
			outLength = int64(count)
		}
		if len(files) == 0 {
			err = io.EOF
		}
	} else {
		outLength = int64(len(files))
	}
	f.readDirCount += outLength

	return files, err
}

func (f *file) Readdirnames(n int) (names []string, err error) {
	fi, err := f.Readdir(n)
	names = make([]string, len(fi))
	for i, f := range fi {
		names[i] = f.Name()
	}
	return names, err
}

func (f *file) Read(buf []byte) (int, error) {
	f.fileData.Lock()
	defer f.fileData.Unlock()
	n, err := f.read(buf, f.offset, io.ErrUnexpectedEOF)
	f.offset += int64(n)
	return n, err
}

func (f *file) read(b []byte, offset int64, err error) (int, error) {
	if f.closed == true {
		return 0, ErrFileClosed
	}
	if len(b) > 0 && int(offset) == len(f.fileData.data) {
		return 0, io.EOF
	}
	if int(f.offset) > len(f.fileData.data) {
		return 0, err
	}
	n := len(b)
	if len(f.fileData.data)-int(offset) < len(b) {
		n = len(f.fileData.data) - int(offset)
	}
	copy(b, f.fileData.data[offset:offset+int64(n)])
	return n, nil
}

func (f *file) ReadAt(b []byte, off int64) (n int, err error) {
	f.fileData.Lock()
	defer f.fileData.Unlock()
	return f.read(b, off, io.EOF)
}

func (f *file) Truncate(size int64) error {
	if f.readOnly {
		return ErrReadOnly
	}
	if f.closed == true {
		return ErrFileClosed
	}
	if size < 0 {
		return ErrOutOfRange
	}
	f.fileData.Lock()
	defer f.fileData.Unlock()
	if size > int64(len(f.fileData.data)) {
		diff := size - int64(len(f.fileData.data))
		f.fileData.data = append(f.fileData.data, bytes.Repeat([]byte{00}, int(diff))...)
	} else {
		f.fileData.data = f.fileData.data[0:size]
	}
	f.fileData.setModTime(time.Now())
	return nil
}

func (f *file) Seek(offset int64, whence int) (int64, error) {
	if f.closed == true {
		return 0, ErrFileClosed
	}
	f.fileData.Lock()
	defer f.fileData.Unlock()
	switch whence {
	case 0:
	case 1:
		offset += f.offset
	case 2:
		offset = int64(len(f.fileData.data)) + offset
	}
	if offset < 0 || offset >= int64(len(f.fileData.data)) {
		return 0, ErrOutOfRange
	}
	f.offset = offset
	return f.offset, nil
}

func (f *file) Write(buf []byte) (int, error) {
	f.fileData.Lock()
	defer f.fileData.Unlock()
	n, err := f.write(buf, f.offset)
	if err != nil {
		return 0, err
	}
	f.offset += int64(n)
	return int(n), nil
}

func (f *file) write(buf []byte, offset int64) (int, error) {
	if f.readOnly {
		return 0, ErrReadOnly
	}
	if f.closed == true {
		return 0, ErrFileClosed
	}
	n := int64(len(buf))
	add := offset + n - int64(len(f.fileData.data))
	copy(f.fileData.data[offset:], buf)
	if add > 0 {
		f.fileData.data = append(f.fileData.data, buf[n-add:]...)
	}
	f.fileData.setModTime(time.Now())
	return int(n), nil
}

func (f *file) WriteAt(buf []byte, off int64) (n int, err error) {
	f.fileData.Lock()
	defer f.fileData.Unlock()
	return f.write(buf, off)
}

func (f *file) WriteString(s string) (ret int, err error) {
	return f.Write([]byte(s))
}

var (
	ErrNoDir      = errors.New("file is no dir")
	ErrReadOnly   = errors.New("filehandle is not writable")
	ErrFileClosed = errors.New("file is closed")
	ErrOutOfRange = errors.New("out of range")
	ErrTooLarge   = errors.New("too large")
	ErrNotEmpty   = errors.New("dir not empty")
)
