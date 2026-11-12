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

package layerfs

import (
	"os"

	. "github.com/mandelsoft/vfs/pkg/test"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
)

var DefaultContent = []byte("This is a test\n")

func NewTestEnv() (vfs.FileSystem, vfs.FileSystem, vfs.FileSystem) {
	layer := memoryfs.New()
	base := memoryfs.New()

	base.MkdirAll("base/d1", os.ModePerm)
	f, err := base.Create("base/d1/basefile")
	if err == nil {
		f.Write(DefaultContent)
		f.Close()
	}
	f, err = base.Create("base/d1/otherfile")
	if err == nil {
		f.Write(DefaultContent)
		f.Close()
	}
	f, err = base.Create("base/basefile")
	if err == nil {
		f.Write(DefaultContent)
		f.Write(DefaultContent)
		f.Close()
	}
	return New(layer, base), layer, base
}

func NewBaseTestEnv() vfs.FileSystem {
	layer := memoryfs.New()
	base := memoryfs.New()
	return New(layer, base)
}

var _ = Describe("layer filesystem", func() {
	StandardTest(NewBaseTestEnv)

	Context("modify", func() {
		var fs vfs.FileSystem
		var layer vfs.FileSystem
		var base vfs.FileSystem

		BeforeEach(func() {
			fs, layer, base = NewTestEnv()
		})

		_, _ = base, layer

		Context("folder", func() {
			It("merge in base layer folder entries", func() {
				ExpectFolders(fs, "base", []string{"basefile", "d1"}, nil)
				ExpectFolders(fs, "base/d1", []string{"basefile", "otherfile"}, nil)
			})
			It("remove entry", func() {
				Expect(fs.Remove("base/basefile")).To(Succeed())
				ExpectFolders(fs, "base", []string{"d1"}, nil)
				ExpectFolders(layer, "base", []string{".wh.basefile"}, nil)
				ExpectFolders(base, "base", []string{"basefile", "d1"}, nil)
			})
			It("recreate entry", func() {
				content := []byte("other content")
				Expect(fs.Remove("base/basefile")).To(Succeed())
				ExpectFileCreate(fs, "base/basefile", content, nil)

				ExpectFolders(fs, "base", []string{"basefile", "d1"}, nil)
				ExpectFolders(layer, "base", []string{"basefile"}, nil)
				ExpectFolders(base, "base", []string{"basefile", "d1"}, nil)
				ExpectFileContent(base, "base/basefile", append(DefaultContent, DefaultContent...))
			})
			It("redelete entry", func() {
				content := []byte("other content")
				Expect(fs.Remove("base/basefile")).To(Succeed())
				ExpectFileCreate(fs, "base/basefile", content, nil)
				Expect(fs.Remove("base/basefile")).To(Succeed())

				ExpectFolders(fs, "base", []string{"d1"}, nil)
				ExpectFolders(layer, "base", []string{".wh.basefile"}, nil)
				ExpectFolders(base, "base", []string{"basefile", "d1"}, nil)
			})

			It("remove base dir", func() {
				Expect(fs.RemoveAll("base/d1")).To(Succeed())
				ExpectFolders(fs, "base", []string{"basefile"}, nil)
				ExpectFolders(layer, "base", []string{".wh.d1"}, nil)
				ExpectFolders(base, "base/d1", []string{"basefile", "otherfile"}, nil)
			})
			It("indirect", func() {
				Expect(fs.Mkdir("d1", os.ModePerm)).To(Succeed())
				Expect(fs.RemoveAll("base")).To(Succeed())
				ExpectFolders(fs, "/", []string{"d1"}, nil)
				ExpectFolders(layer, "/", []string{".wh.base", "d1"}, nil)
				Expect(fs.Mkdir("base", os.ModePerm)).To(Succeed())
				ExpectFolders(fs, "/", []string{"base", "d1"}, nil)
				ExpectFolders(layer, "/", []string{"base", "d1"}, nil)
				ExpectFolders(fs, "/base", []string{}, nil)
				ExpectFolders(layer, "/base", []string{".wh..wh..opq"}, nil)
				Expect(fs.Mkdir("base/d1", os.ModePerm)).To(Succeed())
				ExpectFolders(fs, "/base/d1", []string{}, nil)
				ExpectFolders(base, "/base/d1", []string{"basefile", "otherfile"}, nil)
				ExpectFolders(layer, "/base/d1", []string{".wh..wh..opq"}, nil)
			})
		})
		Context("file", func() {
			It("show file from base", func() {
				ExpectFileContent(fs, "base/d1/basefile", DefaultContent)
			})
			It("overwrite file from base", func() {
				content := []byte("other content")
				ExpectFileWrite(fs, "base/d1/basefile", os.O_CREATE|os.O_TRUNC, content)
				ExpectFileContent(base, "base/d1/basefile", DefaultContent)
				ExpectFolders(fs, "base/d1", []string{"basefile", "otherfile"}, nil)
				ExpectFolders(fs, "base", []string{"basefile", "d1"}, nil)
				ExpectFolders(layer, "base", []string{"d1"}, nil)
				ExpectFolders(layer, "base/d1", []string{"basefile"}, nil)
			})
			It("modify file from base", func() {
				content := []byte("other content")
				ExpectFileWrite(fs, "base/d1/basefile", os.O_CREATE, content, false)
				modified := append(DefaultContent[:0:0], DefaultContent...)
				copy(modified, content)
				ExpectFileContent(fs, "base/d1/basefile", modified)
				ExpectFileContent(base, "base/d1/basefile", DefaultContent)
				ExpectFolders(fs, "base/d1", []string{"basefile", "otherfile"}, nil)
				ExpectFolders(fs, "base", []string{"basefile", "d1"}, nil)
				ExpectFolders(layer, "base", []string{"d1"}, nil)
				ExpectFolders(layer, "base/d1", []string{"basefile"}, nil)
			})
		})
	})

	Context("whiteouts", func() {
		var base, layer vfs.FileSystem
		var fs vfs.FileSystem

		BeforeEach(func() {
			base, layer = memoryfs.New(), memoryfs.New()
			Expect(vfs.WriteFile(base, "/both", []byte("base"), 0o644)).To(Succeed())
			Expect(base.MkdirAll("/dir/sub", 0o755)).To(Succeed())
			Expect(vfs.WriteFile(base, "/dir/sub/x", []byte("x"), 0o644)).To(Succeed())
			fs = New(layer, base)
			Expect(vfs.WriteFile(fs, "/both", []byte("layer"), 0o644)).To(Succeed())
			Expect(vfs.WriteFile(fs, "/only", []byte("only"), 0o644)).To(Succeed())
		})

		It("marks only deletions of entries of the base by Remove", func() {
			Expect(fs.Remove("/both")).To(Succeed())
			Expect(fs.Remove("/only")).To(Succeed())
			Expect(vfs.Exists(fs, "/both")).To(BeFalse())
			Expect(vfs.Exists(fs, "/only")).To(BeFalse())
			ExpectFolders(layer, "/", []string{".wh.both"}, nil)
		})

		It("marks only deletions of entries of the base by RemoveAll", func() {
			Expect(fs.RemoveAll("/both")).To(Succeed())
			Expect(fs.RemoveAll("/only")).To(Succeed())
			// the entry of the base must not appear again
			Expect(vfs.Exists(fs, "/both")).To(BeFalse())
			Expect(vfs.Exists(fs, "/only")).To(BeFalse())
			ExpectFolders(layer, "/", []string{".wh.both"}, nil)
		})

		It("does not mark entries hidden by opaque directories", func() {
			Expect(fs.RemoveAll("/dir")).To(Succeed())
			Expect(fs.MkdirAll("/dir", 0o755)).To(Succeed())
			Expect(vfs.WriteFile(fs, "/dir/sub", []byte("file"), 0o644)).To(Succeed())
			Expect(fs.Remove("/dir/sub")).To(Succeed())
			Expect(vfs.Exists(fs, "/dir/sub")).To(BeFalse())
			ExpectFolders(layer, "/dir", []string{".wh..wh..opq"}, nil)
		})
	})

	It("provides the layer itself without base", func() {
		layer := memoryfs.New()
		Expect(New(layer, nil)).To(BeIdenticalTo(layer))
		var typed *LayerFileSystem
		Expect(New(layer, typed)).To(BeIdenticalTo(layer))
	})

	Context("opaque directories of OCI layers", func() {
		// the layer has only an opaque marker for the top directory, like
		// OCI layers, not for its sub directories
		var base, layer vfs.FileSystem
		var fs vfs.FileSystem

		BeforeEach(func() {
			base, layer = memoryfs.New(), memoryfs.New()
			Expect(base.MkdirAll("/dir/sub", 0o755)).To(Succeed())
			Expect(vfs.WriteFile(base, "/dir/sub/y", []byte("base y"), 0o644)).To(Succeed())
			Expect(vfs.WriteFile(base, "/dir/z", []byte("base z"), 0o644)).To(Succeed())
			Expect(layer.MkdirAll("/dir/sub", 0o755)).To(Succeed())
			Expect(vfs.WriteFile(layer, "/dir/.wh..wh..opq", nil, 0o644)).To(Succeed())
			Expect(vfs.WriteFile(layer, "/dir/sub/x", []byte("layer x"), 0o644)).To(Succeed())
			fs = New(layer, base)
		})

		It("hides the complete sub tree of the base", func() {
			Expect(vfs.Exists(fs, "/dir/z")).To(BeFalse())
			Expect(vfs.Exists(fs, "/dir/sub/y")).To(BeFalse())
			Expect(vfs.ReadFile(fs, "/dir/sub/x")).To(Equal([]byte("layer x")))
			ExpectFolders(fs, "/dir", []string{"sub"}, nil)
			ExpectFolders(fs, "/dir/sub", []string{"x"}, nil)
		})

		It("hides the sub tree of entries listed before the opaque marker", func() {
			// "+" is sorted before "." of the marker
			Expect(base.MkdirAll("/dir/+sub", 0o755)).To(Succeed())
			Expect(vfs.WriteFile(base, "/dir/+sub/y", []byte("base y"), 0o644)).To(Succeed())
			Expect(layer.MkdirAll("/dir/+sub", 0o755)).To(Succeed())
			Expect(vfs.Exists(fs, "/dir/+sub/y")).To(BeFalse())
			ExpectFolders(fs, "/dir/+sub", []string{}, nil)
		})

		It("deletes entries of the sub tree without whiteouts", func() {
			Expect(fs.Remove("/dir/sub/x")).To(Succeed())
			Expect(vfs.Exists(fs, "/dir/sub/x")).To(BeFalse())
			Expect(vfs.Exists(fs, "/dir/sub/y")).To(BeFalse())
			ExpectFolders(layer, "/dir/sub", []string{}, nil)
		})

		It("creates entries in the sub tree without the content of the base", func() {
			Expect(vfs.WriteFile(fs, "/dir/sub/y", []byte("new y"), 0o644)).To(Succeed())
			Expect(vfs.ReadFile(fs, "/dir/sub/y")).To(Equal([]byte("new y")))
			ExpectFolders(fs, "/dir/sub", []string{"x", "y"}, nil)
			Expect(vfs.ReadFile(base, "/dir/sub/y")).To(Equal([]byte("base y")))
		})
	})
})
