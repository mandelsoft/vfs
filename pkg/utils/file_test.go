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

package utils_test

import (
	"io"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/mandelsoft/vfs/pkg/utils"
	"github.com/mandelsoft/vfs/pkg/vfs"
)

var _ = Describe("memory file", func() {
	var fs vfs.FileSystem
	var files []vfs.File

	BeforeEach(func() {
		fs = memoryfs.New()
		files = nil
		Expect(vfs.WriteFile(fs, "file", []byte("abc"), 0o600)).To(Succeed())
	})

	AfterEach(func() {
		for _, f := range files {
			f.Close()
		}
	})

	open := func(flags int) vfs.File {
		f, err := fs.OpenFile("file", flags, 0o600)
		Expect(err).To(Succeed())
		files = append(files, f)
		return f
	}

	content := func() string {
		data, err := vfs.ReadFile(fs, "file")
		Expect(err).To(Succeed())
		return string(data)
	}

	Context("seek", func() {
		It("seeks relative to the start", func() {
			f := open(vfs.O_RDONLY)
			Expect(f.Seek(1, io.SeekStart)).To(Equal(int64(1)))
			Expect(f.Seek(1, io.SeekStart)).To(Equal(int64(1)))
		})

		It("seeks relative to the current offset", func() {
			f := open(vfs.O_RDONLY)
			Expect(f.Seek(1, io.SeekCurrent)).To(Equal(int64(1)))
			Expect(f.Seek(1, io.SeekCurrent)).To(Equal(int64(2)))
			Expect(f.Seek(-2, io.SeekCurrent)).To(Equal(int64(0)))
		})

		It("seeks relative to the end", func() {
			f := open(vfs.O_RDONLY)
			Expect(f.Seek(0, io.SeekEnd)).To(Equal(int64(3)))
			Expect(f.Seek(-1, io.SeekEnd)).To(Equal(int64(2)))
		})

		It("seeks to the end of an empty file", func() {
			Expect(vfs.WriteFile(fs, "file", nil, 0o600)).To(Succeed())
			f := open(vfs.O_RDONLY)
			Expect(f.Seek(0, io.SeekEnd)).To(Equal(int64(0)))
		})

		It("seeks after the end without changing the file", func() {
			f := open(vfs.O_RDONLY)
			Expect(f.Seek(5, io.SeekEnd)).To(Equal(int64(8)))
			fi, err := f.Stat()
			Expect(err).To(Succeed())
			Expect(fi.Size()).To(Equal(int64(3)))
		})

		It("rejects a negative offset and keeps the actual offset", func() {
			f := open(vfs.O_RDONLY)
			Expect(f.Seek(1, io.SeekStart)).To(Equal(int64(1)))
			_, err := f.Seek(-4, io.SeekEnd)
			Expect(err).To(MatchError(utils.ErrOutOfRange))
			Expect(f.Seek(0, io.SeekCurrent)).To(Equal(int64(1)))
		})

		It("rejects an invalid whence", func() {
			f := open(vfs.O_RDONLY)
			_, err := f.Seek(0, 5)
			Expect(err).To(MatchError(os.ErrInvalid))
		})

		It("rejects a closed file", func() {
			f := open(vfs.O_RDONLY)
			Expect(f.Close()).To(Succeed())
			_, err := f.Seek(0, io.SeekStart)
			Expect(err).To(MatchError(utils.ErrFileClosed))
		})
	})

	Context("write", func() {
		It("fills the gap after the end with zeros", func() {
			f := open(vfs.O_RDWR)
			Expect(f.Seek(2, io.SeekEnd)).To(Equal(int64(5)))
			Expect(f.Write([]byte("X"))).To(Equal(1))
			Expect(content()).To(Equal("abc\x00\x00X"))
		})

		It("does not extend the file by an empty write", func() {
			f := open(vfs.O_RDWR)
			Expect(f.Seek(2, io.SeekEnd)).To(Equal(int64(5)))
			Expect(f.Write(nil)).To(Equal(0))
			Expect(content()).To(Equal("abc"))
		})

		It("fills the gap with zeros for WriteAt", func() {
			f := open(vfs.O_RDWR)
			Expect(f.WriteAt([]byte("X"), 4)).To(Equal(1))
			Expect(content()).To(Equal("abc\x00X"))
		})

		It("overwrites and extends", func() {
			f := open(vfs.O_RDWR)
			Expect(f.WriteAt([]byte("XYZ"), 1)).To(Equal(3))
			Expect(content()).To(Equal("aXYZ"))
		})

		It("rejects a negative offset for WriteAt", func() {
			f := open(vfs.O_RDWR)
			_, err := f.WriteAt([]byte("X"), -1)
			Expect(err).To(MatchError(utils.ErrOutOfRange))
		})
	})

	Context("read", func() {
		It("reports EOF at the end", func() {
			f := open(vfs.O_RDONLY)
			buf := make([]byte, 5)
			Expect(f.Read(buf)).To(Equal(3))
			n, err := f.Read(buf)
			Expect(n).To(Equal(0))
			Expect(err).To(Equal(io.EOF))
		})

		It("reports EOF after the end", func() {
			f := open(vfs.O_RDONLY)
			Expect(f.Seek(5, io.SeekStart)).To(Equal(int64(5)))
			n, err := f.Read(make([]byte, 1))
			Expect(n).To(Equal(0))
			Expect(err).To(Equal(io.EOF))
		})

		It("reads nothing without error for an empty buffer", func() {
			f := open(vfs.O_RDONLY)
			Expect(f.Seek(5, io.SeekStart)).To(Equal(int64(5)))
			Expect(f.Read(nil)).To(Equal(0))
		})

		It("reads at the given offset independent of the actual offset", func() {
			f := open(vfs.O_RDONLY)
			Expect(f.Seek(5, io.SeekStart)).To(Equal(int64(5)))
			buf := make([]byte, 2)
			Expect(f.ReadAt(buf, 1)).To(Equal(2))
			Expect(string(buf)).To(Equal("bc"))
		})

		It("reports EOF for a partial ReadAt", func() {
			f := open(vfs.O_RDONLY)
			buf := make([]byte, 5)
			n, err := f.ReadAt(buf, 1)
			Expect(n).To(Equal(2))
			Expect(err).To(Equal(io.EOF))
			Expect(string(buf[:n])).To(Equal("bc"))
		})

		It("rejects a negative offset for ReadAt", func() {
			f := open(vfs.O_RDONLY)
			_, err := f.ReadAt(make([]byte, 1), -1)
			Expect(err).To(MatchError(utils.ErrOutOfRange))
		})
	})

	Context("append mode", func() {
		It("appends to an existing file", func() {
			f := open(vfs.O_WRONLY | vfs.O_APPEND)
			Expect(f.Write([]byte("def"))).To(Equal(3))
			Expect(content()).To(Equal("abcdef"))
		})

		It("creates a file to append to", func() {
			f, err := fs.OpenFile("new", vfs.O_WRONLY|vfs.O_CREATE|vfs.O_APPEND, 0o600)
			Expect(err).To(Succeed())
			defer f.Close()
			Expect(f.Write([]byte("def"))).To(Equal(3))
			Expect(vfs.ReadFile(fs, "new")).To(Equal([]byte("def")))
		})

		It("writes at the end after a seek", func() {
			f := open(vfs.O_RDWR | vfs.O_APPEND)
			Expect(f.Seek(1, io.SeekStart)).To(Equal(int64(1)))
			Expect(f.Write([]byte("d"))).To(Equal(1))
			Expect(f.Seek(0, io.SeekCurrent)).To(Equal(int64(4)))
			Expect(content()).To(Equal("abcd"))
		})

		It("writes at the end after a change by another handle", func() {
			f := open(vfs.O_WRONLY | vfs.O_APPEND)
			Expect(vfs.WriteFile(fs, "file", []byte("abcdef"), 0o600)).To(Succeed())
			Expect(f.Write([]byte("g"))).To(Equal(1))
			Expect(content()).To(Equal("abcdefg"))
		})

		It("reads from the start", func() {
			f := open(vfs.O_RDWR | vfs.O_APPEND)
			buf := make([]byte, 3)
			Expect(f.Read(buf)).To(Equal(3))
			Expect(string(buf)).To(Equal("abc"))
		})

		It("rejects WriteAt", func() {
			f := open(vfs.O_RDWR | vfs.O_APPEND)
			_, err := f.WriteAt([]byte("X"), 0)
			Expect(err).To(MatchError(utils.ErrWriteAtInAppendMode))
			Expect(content()).To(Equal("abc"))
		})
	})
})
