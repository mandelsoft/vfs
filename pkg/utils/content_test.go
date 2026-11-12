/*
 * Copyright 2026 Mandelsoft. All rights reserved.
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
	"bytes"
	"errors"
	"io"
	"os"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/mandelsoft/vfs/pkg/utils"
	"github.com/mandelsoft/vfs/pkg/vfs"
)

// readerKind selects the kind of reader provided by stream content.
type readerKind int

const (
	plainReader readerKind = iota
	seekReader
	readerAtReader
)

// streamData is a test implementation of FileData for directories and
// files with stream content.
type streamData struct {
	sync.Mutex
	mode    os.FileMode
	modtime time.Time
	entries map[string]utils.FileData

	content  []byte
	kind     readerKind
	readOnly bool
	// once marks content, which can only be read once.
	once bool

	// opened and closed count the readers, memory counts calls of Data
	// and SetData, which must not happen for stream content.
	opened, closed int
	memory         int
}

var _ utils.StreamData = (*streamData)(nil)

func newDir() *streamData {
	return &streamData{mode: os.ModeDir | 0o755, entries: map[string]utils.FileData{}}
}

func newStream(content string, kind readerKind) *streamData {
	return &streamData{mode: 0o644, content: []byte(content), kind: kind}
}

func (d *streamData) IsDir() bool        { return d.mode.IsDir() }
func (d *streamData) IsSymlink() bool    { return false }
func (d *streamData) GetSymlink() string { return "" }

func (d *streamData) GetEntry(name string) (utils.FileDataDirAccess, error) {
	if e, ok := d.entries[name]; ok {
		return e, nil
	}
	return nil, os.ErrNotExist
}

func (d *streamData) Data() []byte           { d.memory++; return nil }
func (d *streamData) SetData([]byte)         { d.memory++ }
func (d *streamData) Mode() os.FileMode      { return d.mode }
func (d *streamData) SetMode(m os.FileMode)  { d.mode = m }
func (d *streamData) ModTime() time.Time     { return d.modtime }
func (d *streamData) SetModTime(t time.Time) { d.modtime = t }

func (d *streamData) Files() []os.FileInfo {
	var list []os.FileInfo
	for n, e := range d.entries {
		list = append(list, utils.NewFileInfo(n, e))
	}
	return list
}

func (d *streamData) Add(name string, f utils.FileData) error {
	if _, ok := d.entries[name]; ok {
		return os.ErrExist
	}
	d.entries[name] = f
	return nil
}

func (d *streamData) Del(name string) error {
	delete(d.entries, name)
	return nil
}

func (d *streamData) Size() int64 {
	return int64(len(d.content))
}

func (d *streamData) Reader() (io.ReadCloser, error) {
	if d.once && d.opened > 0 {
		return nil, errors.New("already read")
	}
	d.opened++
	r := bytes.NewReader(d.content)
	switch d.kind {
	case seekReader:
		return &seekCloser{readCloser{r, d}, r}, nil
	case readerAtReader:
		return &readerAtCloser{readCloser{r, d}, r}, nil
	default:
		return &readCloser{r, d}, nil
	}
}

func (d *streamData) Writer() (io.WriteCloser, error) {
	if d.readOnly {
		return nil, vfs.ErrReadOnly
	}
	return &writeCloser{&bytes.Buffer{}, d}, nil
}

// readCloser is a plain reader counting its closing.
type readCloser struct {
	r io.Reader
	d *streamData
}

func (r *readCloser) Read(p []byte) (int, error) { return r.r.Read(p) }
func (r *readCloser) Close() error               { r.d.closed++; return nil }

type seekCloser struct {
	readCloser
	s io.Seeker
}

func (r *seekCloser) Seek(offset int64, whence int) (int64, error) {
	return r.s.Seek(offset, whence)
}

type readerAtCloser struct {
	readCloser
	ra io.ReaderAt
}

func (r *readerAtCloser) ReadAt(p []byte, off int64) (int, error) {
	return r.ra.ReadAt(p, off)
}

// writeCloser sets the content on closing.
type writeCloser struct {
	*bytes.Buffer
	d *streamData
}

func (w *writeCloser) Close() error {
	w.d.content = w.Bytes()
	return nil
}

// adapter creates stream content for new files.
type adapter struct{}

func (adapter) CreateFile(perm os.FileMode) utils.FileData {
	return &streamData{mode: perm, kind: plainReader}
}

func (adapter) CreateDir(perm os.FileMode) utils.FileData {
	d := newDir()
	d.mode = os.ModeDir | perm
	return d
}

func (adapter) CreateSymlink(string, os.FileMode) utils.FileData {
	return nil
}

var _ = Describe("stream content", func() {
	var root *streamData
	var fs vfs.FileSystem

	BeforeEach(func() {
		root = newDir()
		fs = utils.NewFSSupport("stream", root, adapter{})
	})

	add := func(name string, d *streamData) *streamData {
		Expect(root.Add(name, d)).To(Succeed())
		return d
	}

	readAt := func(f vfs.File, off int64, n int) string {
		buf := make([]byte, n)
		r, err := f.ReadAt(buf, off)
		Expect(err).To(Succeed())
		return string(buf[:r])
	}

	read := func(f vfs.File, n int) string {
		buf := make([]byte, n)
		r, err := io.ReadFull(f, buf)
		Expect(err).To(Succeed())
		return string(buf[:r])
	}

	Context("reading", func() {
		It("provides the size and the content without using memory data", func() {
			d := add("file", newStream("0123456789", plainReader))
			info, err := fs.Stat("file")
			Expect(err).To(Succeed())
			Expect(info.Size()).To(Equal(int64(10)))
			Expect(vfs.ReadFile(fs, "file")).To(Equal([]byte("0123456789")))
			Expect(d.opened).To(Equal(1))
			Expect(d.closed).To(Equal(1))
			Expect(d.memory).To(Equal(0))
		})

		It("skips forward and reads again from the beginning for backward positions", func() {
			d := add("file", newStream("0123456789", plainReader))
			f, err := fs.Open("file")
			Expect(err).To(Succeed())
			defer f.Close()

			Expect(read(f, 2)).To(Equal("01"))
			Expect(f.Seek(5, io.SeekStart)).To(Equal(int64(5)))
			Expect(read(f, 2)).To(Equal("56"))
			Expect(d.opened).To(Equal(1))

			Expect(f.Seek(-1, io.SeekEnd)).To(Equal(int64(9)))
			Expect(read(f, 1)).To(Equal("9"))
			Expect(f.Seek(1, io.SeekStart)).To(Equal(int64(1)))
			Expect(read(f, 3)).To(Equal("123"))
			Expect(d.opened).To(Equal(2))
			Expect(d.closed).To(Equal(1))
		})

		It("cannot read backwards for content readable only once", func() {
			d := add("file", newStream("0123456789", plainReader))
			d.once = true
			f, err := fs.Open("file")
			Expect(err).To(Succeed())
			defer f.Close()

			Expect(read(f, 4)).To(Equal("0123"))
			Expect(f.Seek(1, io.SeekStart)).To(Equal(int64(1)))
			_, err = f.Read(make([]byte, 1))
			Expect(err).To(MatchError(errors.ErrUnsupported))
			Expect(err).To(MatchError(ContainSubstring("cannot read the content again: already read")))

			// the actual reader is still usable
			Expect(f.Seek(6, io.SeekStart)).To(Equal(int64(6)))
			Expect(read(f, 2)).To(Equal("67"))
			Expect(d.closed).To(Equal(0))
		})

		It("uses seekable readers for positioning", func() {
			d := add("file", newStream("0123456789", seekReader))
			f, err := fs.Open("file")
			Expect(err).To(Succeed())
			defer f.Close()

			Expect(readAt(f, 7, 2)).To(Equal("78"))
			Expect(readAt(f, 1, 2)).To(Equal("12"))
			Expect(d.opened).To(Equal(1))
		})

		It("uses readers with direct access", func() {
			d := add("file", newStream("0123456789", readerAtReader))
			f, err := fs.Open("file")
			Expect(err).To(Succeed())
			defer f.Close()

			Expect(readAt(f, 7, 2)).To(Equal("78"))
			Expect(readAt(f, 1, 2)).To(Equal("12"))
			Expect(read(f, 3)).To(Equal("012"))
			Expect(d.opened).To(Equal(1))
		})

		It("reports the end of the content like memory files", func() {
			add("file", newStream("0123456789", plainReader))
			f, err := fs.Open("file")
			Expect(err).To(Succeed())
			defer f.Close()

			buf := make([]byte, 4)
			n, err := f.ReadAt(buf, 8)
			Expect(n).To(Equal(2))
			Expect(err).To(MatchError(io.EOF))
			_, err = f.ReadAt(buf, 10)
			Expect(err).To(MatchError(io.EOF))

			Expect(f.Seek(8, io.SeekStart)).To(Equal(int64(8)))
			n, err = f.Read(buf)
			Expect(n).To(Equal(2))
			Expect(err).To(Succeed())
			_, err = f.Read(buf)
			Expect(err).To(MatchError(io.EOF))
		})

		It("closes the reader with the file", func() {
			d := add("file", newStream("0123456789", plainReader))
			f, err := fs.Open("file")
			Expect(err).To(Succeed())
			Expect(read(f, 2)).To(Equal("01"))
			Expect(f.Close()).To(Succeed())
			Expect(d.closed).To(Equal(1))
		})
	})

	Context("writing", func() {
		It("replaces the content of truncated files", func() {
			d := add("file", newStream("0123456789", plainReader))
			f, err := fs.OpenFile("file", vfs.O_WRONLY|vfs.O_TRUNC, 0)
			Expect(err).To(Succeed())
			Expect(f.Write([]byte("abc"))).To(Equal(3))
			Expect(f.Write([]byte("de"))).To(Equal(2))

			info, err := f.Stat()
			Expect(err).To(Succeed())
			Expect(info.Size()).To(Equal(int64(10))) // completed by closing
			Expect(f.Seek(0, io.SeekEnd)).To(Equal(int64(5)))

			Expect(f.Close()).To(Succeed())
			Expect(string(d.content)).To(Equal("abcde"))
			Expect(vfs.ReadFile(fs, "file")).To(Equal([]byte("abcde")))
			Expect(d.memory).To(Equal(0))
		})

		It("writes new files", func() {
			Expect(vfs.WriteFile(fs, "new", []byte("content"), 0o644)).To(Succeed())
			Expect(vfs.ReadFile(fs, "new")).To(Equal([]byte("content")))
		})

		It("provides empty content for files truncated without writing", func() {
			d := add("file", newStream("0123456789", plainReader))
			f, err := fs.OpenFile("file", vfs.O_RDWR, 0)
			Expect(err).To(Succeed())
			Expect(f.Truncate(10)).To(Succeed())
			Expect(f.Truncate(0)).To(Succeed())
			Expect(f.Close()).To(Succeed())
			Expect(d.content).To(BeEmpty())
		})

		It("rejects other write operations", func() {
			add("file", newStream("0123456789", plainReader))
			f, err := fs.OpenFile("file", vfs.O_RDWR, 0)
			Expect(err).To(Succeed())
			defer f.Close()

			_, err = f.Write([]byte("x"))
			Expect(err).To(MatchError(errors.ErrUnsupported))
			Expect(err).To(MatchError(ContainSubstring("writing without truncating the content")))

			err = f.Truncate(5)
			Expect(err).To(MatchError(errors.ErrUnsupported))
			Expect(err).To(MatchError(ContainSubstring("truncating to size 5")))

			Expect(f.Truncate(0)).To(Succeed())
			_, err = f.WriteAt([]byte("x"), 2)
			Expect(err).To(MatchError(errors.ErrUnsupported))
			Expect(err).To(MatchError(ContainSubstring("writing at offset 2 instead of 0")))

			_, err = f.ReadAt(make([]byte, 1), 0)
			Expect(err).To(MatchError(errors.ErrUnsupported))
			Expect(err).To(MatchError(ContainSubstring("reading content being replaced")))
		})

		It("reports read-only content", func() {
			d := add("file", newStream("0123456789", plainReader))
			d.readOnly = true
			f, err := fs.OpenFile("file", vfs.O_WRONLY|vfs.O_TRUNC, 0)
			Expect(err).To(Succeed())
			_, err = f.Write([]byte("x"))
			Expect(err).To(MatchError(vfs.ErrReadOnly))
			Expect(f.Close()).To(MatchError(vfs.ErrReadOnly))
			Expect(string(d.content)).To(Equal("0123456789"))
		})
	})
})
