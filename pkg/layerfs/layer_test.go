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

package layerfs

import (
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/mandelsoft/vfs/pkg/memoryfs"
	. "github.com/mandelsoft/vfs/pkg/test"
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
})
