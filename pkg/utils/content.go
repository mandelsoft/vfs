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

package utils

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"time"
)

// StreamData is an optional interface of FileData, providing the content
// of a file by streams instead of Data and SetData. It is used for content,
// which should not be kept in memory, for example content read on demand
// from another storage.
//
// Files with stream content support reading with arbitrary offsets: a
// reader is read sequentially, positioning backwards opens a new reader (if
// possible), readers implementing io.Seeker or io.ReaderAt are used for
// direct access.
// Writing is only supported as sequential replacement of the complete
// content: after truncating the file to size 0 (for example by opening it
// with O_TRUNC) or for empty content, the content is written from offset 0
// without gaps. The new content is complete, when the file is closed.
// Other write operations fail with an error matching errors.ErrUnsupported.
type StreamData interface {
	FileData
	// Size provides the size of the content.
	Size() int64
	// Reader provides a new reader for the content from its beginning.
	// Content, which can only be read once, provides an error for further
	// readers, positioning backwards is not possible then.
	Reader() (io.ReadCloser, error)
	// Writer provides a writer replacing the complete content. Closing it
	// completes the new content. Read-only content provides an error
	// (vfs.ErrReadOnly).
	Writer() (io.WriteCloser, error)
}

// contentHandle is the access to the content of a file used by a file
// handle. The file data is locked by the caller.
type contentHandle interface {
	// size provides the actual size of the content.
	size() int64
	// read reads at most len(b) bytes at the given offset. It provides
	// io.EOF, if the offset is at or after the end of the content.
	read(b []byte, offset int64) (int, error)
	// write writes the data at the given offset.
	write(buf []byte, offset int64) (int, error)
	// truncate changes the size of the content.
	truncate(size int64) error
	// close releases the resources of the handle.
	close() error
}

// newContent provides the content handle for some file data.
func newContent(data FileData) contentHandle {
	if s, ok := data.(StreamData); ok {
		return &streamContent{data: s}
	}
	return &memoryContent{data}
}

////////////////////////////////////////////////////////////////////////////////

// memoryContent is the content kept in memory by the file data.
type memoryContent struct {
	data FileData
}

var _ contentHandle = (*memoryContent)(nil)

func (c *memoryContent) size() int64 {
	return int64(len(c.data.Data()))
}

func (c *memoryContent) read(b []byte, offset int64) (int, error) {
	data := c.data.Data()
	if offset >= int64(len(data)) {
		if len(b) > 0 {
			return 0, io.EOF
		}
		return 0, nil
	}
	return copy(b, data[offset:]), nil
}

// write writes data at the given offset. If the offset is after the end
// of the file, the gap is filled with zeros.
func (c *memoryContent) write(buf []byte, offset int64) (int, error) {
	data := c.data.Data()
	if gap := offset - int64(len(data)); gap > 0 {
		data = append(data, make([]byte, gap)...)
	}
	n := int64(len(buf))
	add := offset + n - int64(len(data))
	copy(data[offset:], buf)
	if add > 0 {
		data = append(data, buf[n-add:]...)
	}
	c.data.SetData(data)
	c.data.SetModTime(time.Now())
	return int(n), nil
}

func (c *memoryContent) truncate(size int64) error {
	data := c.data.Data()
	if size > int64(len(data)) {
		diff := size - int64(len(data))
		c.data.SetData(append(data, bytes.Repeat([]byte{00}, int(diff))...))
	} else {
		c.data.SetData(data[0:size])
	}
	c.data.SetModTime(time.Now())
	return nil
}

func (c *memoryContent) close() error {
	return nil
}

////////////////////////////////////////////////////////////////////////////////

// streamContent is the content provided by streams (see StreamData).
type streamContent struct {
	data StreamData

	// reader is the actual reader and pos its position.
	reader io.ReadCloser
	pos    int64

	// replacing indicates that the content is being replaced, written is
	// the size of the new content written so far by writer.
	replacing bool
	writer    io.WriteCloser
	written   int64
}

var _ contentHandle = (*streamContent)(nil)

func unsupported(format string, args ...any) error {
	return fmt.Errorf("%s: %w", fmt.Sprintf(format, args...), errors.ErrUnsupported)
}

// join combines two errors, a single error is returned as it is.
func join(a, b error) error {
	switch {
	case a == nil:
		return b
	case b == nil:
		return a
	}
	return errors.Join(a, b)
}

func (c *streamContent) size() int64 {
	if c.replacing {
		return c.written
	}
	return c.data.Size()
}

func (c *streamContent) read(b []byte, offset int64) (int, error) {
	if c.replacing {
		return 0, unsupported("reading content being replaced")
	}
	if offset >= c.data.Size() {
		if len(b) > 0 {
			return 0, io.EOF
		}
		return 0, nil
	}
	if c.reader == nil {
		r, err := c.data.Reader()
		if err != nil {
			return 0, err
		}
		c.reader, c.pos = r, 0
	}
	if ra, ok := c.reader.(io.ReaderAt); ok {
		n, err := ra.ReadAt(b, offset)
		if errors.Is(err, io.EOF) && n > 0 {
			err = nil
		}
		return n, err
	}
	if err := c.position(offset); err != nil {
		return 0, err
	}
	n, err := io.ReadFull(c.reader, b)
	c.pos += int64(n)
	if errors.Is(err, io.ErrUnexpectedEOF) {
		err = nil
	}
	return n, err
}

// position positions the reader at the given offset: by seeking, by
// skipping forward or by reading again from the beginning.
func (c *streamContent) position(offset int64) error {
	if offset == c.pos {
		return nil
	}
	if s, ok := c.reader.(io.Seeker); ok {
		pos, err := s.Seek(offset, io.SeekStart)
		c.pos = pos
		return err
	}
	if offset < c.pos {
		// reset by a new reader, the actual one is kept, if this fails
		r, err := c.data.Reader()
		if err != nil {
			return fmt.Errorf("%w: cannot read the content again: %w", errors.ErrUnsupported, err)
		}
		err = c.reader.Close()
		c.reader, c.pos = r, 0
		if err != nil {
			return err
		}
	}
	n, err := io.CopyN(io.Discard, c.reader, offset-c.pos)
	c.pos += n
	return err
}

func (c *streamContent) write(buf []byte, offset int64) (int, error) {
	if !c.replacing {
		if c.data.Size() != 0 {
			return 0, unsupported("writing without truncating the content")
		}
		if err := c.startReplacement(); err != nil {
			return 0, err
		}
	}
	if offset != c.written {
		return 0, unsupported("writing at offset %d instead of %d", offset, c.written)
	}
	if c.writer == nil {
		w, err := c.data.Writer()
		if err != nil {
			return 0, err
		}
		c.writer = w
	}
	n, err := c.writer.Write(buf)
	c.written += int64(n)
	return n, err
}

func (c *streamContent) truncate(size int64) error {
	if size == c.size() {
		return nil
	}
	if size != 0 {
		return unsupported("truncating to size %d", size)
	}
	return c.startReplacement()
}

// startReplacement starts a new replacement of the content. A replacement
// already in progress is discarded.
func (c *streamContent) startReplacement() error {
	var err error
	if c.reader != nil {
		err = c.reader.Close()
		c.reader = nil
	}
	if c.writer != nil {
		err = join(err, c.writer.Close())
		c.writer = nil
	}
	c.replacing, c.written = true, 0
	return err
}

func (c *streamContent) close() error {
	var err error
	if c.reader != nil {
		err = c.reader.Close()
		c.reader = nil
	}
	if c.replacing {
		// truncated without writing: the new content is empty
		if c.writer == nil {
			w, werr := c.data.Writer()
			if werr != nil {
				c.replacing = false
				return join(err, werr)
			}
			c.writer = w
		}
		err = join(err, c.writer.Close())
		c.writer, c.replacing = nil, false
		c.data.SetModTime(time.Now())
	}
	return err
}
