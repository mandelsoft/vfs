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
	"bytes"
	"errors"
	"io"
	"io/fs"
	"os"
	"sort"
	"time"

	"github.com/mandelsoft/vfs/pkg/vfs"
)

type FileData interface {
	FileDataDirAccess
	Data() []byte
	Files() []os.FileInfo
	SetData([]byte)
	Mode() os.FileMode
	SetMode(mode os.FileMode)
	ModTime() time.Time
	SetModTime(time.Time)
	Add(name string, f FileData) error
	Del(name string) error
}

type File struct {
	// atomic requires 64-bit alignment for struct field access
	offset       int64
	readDirCount int64
	closed       bool
	readOnly     bool
	// append mode: every Write is done at the end of the file.
	append   bool
	fileData FileData
	name     string
}

var _ vfs.File = &File{}

func newFileHandle(name string, data FileData) *File {
	return &File{name: name, fileData: data}
}

func (f *File) Open() error {
	f.fileData.Lock()
	f.offset = 0
	f.readDirCount = 0
	f.closed = false
	f.fileData.Unlock()
	return nil
}

func (f *File) Close() error {
	f.fileData.Lock()
	f.closed = true
	f.fileData.Unlock()
	return nil
}

func (f *File) Name() string {
	return f.name
}

func (f *File) Stat() (os.FileInfo, error) {
	return NewFileInfo(f.name, f.fileData), nil
}

func (f *File) Sync() error {
	return nil
}

func (f *File) ReadDir(count int) (files []fs.DirEntry, err error) {
	o, err := f.Readdir(count)

	if err != nil {
		return nil, err
	}
	r := make([]fs.DirEntry, len(o), len(o))
	for i, v := range o {
		r[i] = fs.FileInfoToDirEntry(v)
	}
	return r, nil
}

func (f *File) Readdir(count int) (files []os.FileInfo, err error) {
	if !f.fileData.IsDir() {
		return nil, &os.PathError{Op: "readdir", Path: f.name, Err: ErrNotDir}
	}
	var outLength int64

	f.fileData.Lock()
	defer f.fileData.Unlock()

	files = f.fileData.Files()
	sort.Sort(FilesSorter(files))
	if f.readDirCount >= int64(len(files)) {
		files = []os.FileInfo{}
	} else {
		files = files[f.readDirCount:]
	}
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

func (f *File) Readdirnames(n int) (names []string, err error) {
	fi, err := f.Readdir(n)
	names = make([]string, len(fi))
	for i, f := range fi {
		names[i] = f.Name()
	}
	return names, err
}

func (f *File) Read(buf []byte) (int, error) {
	f.fileData.Lock()
	defer f.fileData.Unlock()
	n, err := f.read(buf, f.offset)
	f.offset += int64(n)
	return n, err
}

// read reads data at the given offset. Reading at or after the end of the
// file provides io.EOF (for a non-empty buffer).
func (f *File) read(b []byte, offset int64) (int, error) {
	if f.closed == true {
		return 0, ErrFileClosed
	}
	if offset < 0 {
		return 0, ErrOutOfRange
	}
	data := f.fileData.Data()
	if offset >= int64(len(data)) {
		if len(b) > 0 {
			return 0, io.EOF
		}
		return 0, nil
	}
	return copy(b, data[offset:]), nil
}

// ReadAt reads data at the given offset. According to io.ReaderAt
// it returns io.EOF, if less than len(b) bytes could be read.
func (f *File) ReadAt(b []byte, off int64) (n int, err error) {
	f.fileData.Lock()
	defer f.fileData.Unlock()
	n, err = f.read(b, off)
	if err == nil && n < len(b) {
		err = io.EOF
	}
	return n, err
}

func (f *File) Truncate(size int64) error {
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
	data := f.fileData.Data()
	if size > int64(len(data)) {
		diff := size - int64(len(data))
		f.fileData.SetData(append(data, bytes.Repeat([]byte{00}, int(diff))...))
	} else {
		f.fileData.SetData(data[0:size])
	}
	f.fileData.SetModTime(time.Now())
	return nil
}

func (f *File) Seek(offset int64, whence int) (int64, error) {
	if f.closed == true {
		return 0, ErrFileClosed
	}
	f.fileData.Lock()
	defer f.fileData.Unlock()
	data := f.fileData.Data()
	switch whence {
	case io.SeekStart:
	case io.SeekCurrent:
		offset += f.offset
	case io.SeekEnd:
		offset = int64(len(data)) + offset
	default:
		return 0, &os.PathError{Op: "seek", Path: f.name, Err: os.ErrInvalid}
	}
	// like for OS files, an offset after the end of the file is valid,
	// a following write fills the gap with zeros.
	if offset < 0 {
		return 0, ErrOutOfRange
	}
	f.offset = offset
	return f.offset, nil
}

// Write writes at the actual offset, or in append mode,
// at the end of the file.
func (f *File) Write(buf []byte) (int, error) {
	f.fileData.Lock()
	defer f.fileData.Unlock()
	offset := f.offset
	if f.append {
		offset = int64(len(f.fileData.Data()))
	}
	n, err := f.write(buf, offset)
	if err != nil {
		return 0, err
	}
	f.offset = offset + int64(n)
	return int(n), nil
}

// write writes data at the given offset. If the offset is after the end
// of the file, the gap is filled with zeros.
func (f *File) write(buf []byte, offset int64) (int, error) {
	if f.readOnly {
		return 0, ErrReadOnly
	}
	if f.closed == true {
		return 0, ErrFileClosed
	}
	if offset < 0 {
		return 0, ErrOutOfRange
	}
	if len(buf) == 0 {
		return 0, nil
	}
	data := f.fileData.Data()
	if gap := offset - int64(len(data)); gap > 0 {
		data = append(data, make([]byte, gap)...)
	}
	n := int64(len(buf))
	add := offset + n - int64(len(data))
	copy(data[offset:], buf)
	if add > 0 {
		data = append(data, buf[n-add:]...)
	}
	f.fileData.SetData(data)
	f.fileData.SetModTime(time.Now())
	return int(n), nil
}

// WriteAt writes data at the given offset. Like for OS files,
// it is not possible in append mode.
func (f *File) WriteAt(buf []byte, off int64) (n int, err error) {
	if f.append {
		return 0, ErrWriteAtInAppendMode
	}
	f.fileData.Lock()
	defer f.fileData.Unlock()
	return f.write(buf, off)
}

func (f *File) WriteString(s string) (ret int, err error) {
	return f.Write([]byte(s))
}

var (
	ErrNotDir   = vfs.ErrNotDir
	ErrReadOnly = vfs.ErrReadOnly

	ErrFileClosed = errors.New("file is closed")
	ErrOutOfRange = errors.New("out of range")
	ErrTooLarge   = errors.New("too large")

	ErrWriteAtInAppendMode = errors.New("invalid use of WriteAt on file opened with O_APPEND")
	ErrNotEmpty            = vfs.ErrNotEmpty
)
