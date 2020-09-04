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

package vfs_test

import (
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/mandelsoft/vfs/pkg/cwdfs"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	. "github.com/mandelsoft/vfs/pkg/test"
	. "github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/mandelsoft/vfs/test/pkg/test"
)

var _ = Describe("filesystem", func() {
	var fs *VFS

	BeforeEach(func() {
		fs = New(memoryfs.New())
	})

	Context("utils", func() {
		It("base", func() {
			Expect(fs.Base("/")).To(Equal("/"))
			Expect(fs.Base("//")).To(Equal("/"))
			Expect(fs.Base("")).To(Equal("."))
			Expect(fs.Base(".")).To(Equal("."))
			Expect(fs.Base("/.")).To(Equal("."))
			Expect(fs.Base("/base")).To(Equal("base"))
			Expect(fs.Base("/base/")).To(Equal("base"))
			Expect(fs.Base("/base/.")).To(Equal("."))
			Expect(fs.Base("/path/base/.")).To(Equal("."))
		})

		It("dir/base combinations", func() {
			parts := []string{"file", "/", ".", ".."}
			type check func([]string)

			level := func(f check) check {
				return func(segments []string) {
					for _, s := range parts {
						next := append(segments, s)
						f(next)
					}
				}
			}
			level(level(level(level(func(segments []string) {
				path := Join(nil, segments...)
				//fmt.Printf("trimming %+v:  %q\n", segments, path)
				Expect(Trim(nil, Join(nil, Dir(nil, path), Base(nil, path)))).To(Equal(Trim(nil, path)))
			}))))([]string{})

		})

		It("trim", func() {
			Expect(fs.Trim("path/")).To(Equal("path"))
			Expect(fs.Trim("path//")).To(Equal("path"))
			Expect(fs.Trim("path/other")).To(Equal("path/other"))
			Expect(fs.Trim("path//other")).To(Equal("path/other"))
			Expect(fs.Trim("path//other/")).To(Equal("path/other"))
			Expect(fs.Trim("path//other//")).To(Equal("path/other"))
			Expect(fs.Trim("/path/other/")).To(Equal("/path/other"))
			Expect(fs.Trim("/path/other//")).To(Equal("/path/other"))
			Expect(fs.Trim("//path//other")).To(Equal("/path/other"))
			Expect(fs.Trim("//path//other/")).To(Equal("/path/other"))
			Expect(fs.Trim("//path//other//")).To(Equal("/path/other"))
			Expect(fs.Trim("//")).To(Equal("/"))
			Expect(fs.Trim("/./a/.")).To(Equal("/a"))
			Expect(fs.Trim("/../a/.")).To(Equal("/../a"))
			Expect(fs.Trim("././a/.")).To(Equal("a"))
			Expect(fs.Trim("./../a/.")).To(Equal("../a"))
			Expect(fs.Trim(".")).To(Equal("."))
			Expect(fs.Trim("/.")).To(Equal("/"))
			Expect(fs.Trim("//.")).To(Equal("/"))
			Expect(fs.Trim("//.//")).To(Equal("/"))
		})
		It("join", func() {
			Expect(fs.Join("path")).To(Equal("path"))
			Expect(fs.Join("path", "other")).To(Equal("path/other"))
			Expect(fs.Join("/path", "/other")).To(Equal("/path/other"))
			Expect(fs.Join("", "/other")).To(Equal("/other"))
			Expect(fs.Join("", "other")).To(Equal("other"))
			Expect(fs.Join("", "path", "", "", "other", "")).To(Equal("path/other"))
			Expect(fs.Join("//")).To(Equal("/"))
		})

		Context("ReadFile", func() {
			It("read existing file", func() {
				content := []byte("This is a test")
				test.ExpectCreateFile(fs, "f1", content, nil)
				result, err := ReadFile(fs, "f1")
				Expect(err).To(Succeed())
				Expect(result).To(Equal(content))
			})

			It("read non-existing file", func() {
				result, err := ReadFile(fs, "f1")
				Expect(err).To(HaveOccurred())
				Expect(result).To(HaveLen(0))
			})
		})

		Context("WriteFile", func() {
			It("write non-existing file", func() {
				content := []byte("This is a test")
				Expect(WriteFile(fs, "f1", content, os.ModePerm)).To(Succeed())

				file, err := fs.Open("f1")
				Expect(err).To(Succeed())
				test.ExpectRead(file, content)
				Expect(file.Close()).To(Succeed())
			})

			It("overwrite existing file", func() {
				content := []byte("Other")
				test.ExpectCreateFile(fs, "f1", []byte("This is a test"), nil)
				Expect(WriteFile(fs, "f1", content, os.ModePerm)).To(Succeed())

				file, err := fs.Open("f1")
				Expect(err).To(Succeed())
				test.ExpectRead(file, content)
				Expect(file.Close()).To(Succeed())
			})
		})

		Context("ReadDir", func() {
			It("read all directories of a subpath", func() {
				Expect(fs.MkdirAll("/d1/d11/d111", os.ModePerm)).To(Succeed())
				Expect(fs.MkdirAll("/d1/d12/d112", os.ModePerm)).To(Succeed())

				dirs, err := ReadDir(fs, "/d1")
				Expect(err).To(Succeed())
				Expect(dirs).To(HaveLen(2))
				Expect(dirs[0].IsDir()).To(BeTrue())
				Expect(dirs[0].Name()).To(Equal("d11"))
				Expect(dirs[1].IsDir()).To(BeTrue())
				Expect(dirs[1].Name()).To(Equal("d12"))
			})
		})

	})

	Context("eval sym links", func() {
		var fs *VFS
		var cwd *VFS

		BeforeEach(func() {
			fs = New(memoryfs.New())
			Expect(fs.Mkdir("d1", os.ModePerm)).To(Succeed())
			Expect(fs.Mkdir("d1/d2", os.ModePerm)).To(Succeed())
			Expect(fs.Mkdir("d1/d2/d3", os.ModePerm)).To(Succeed())
			ExpectFileCreate(fs, "d1/d2/f", nil, nil)

			t, _ := cwdfs.New(fs, "/d1")
			cwd = New(t)
		})

		It("link rel back fail", func() {
			Expect(fs.Symlink("f/..", "d1/d2/link")).To(Succeed())
			ExpectErr(EvalSymlinks(fs, "d1/d2/link"))
		})
		It("rel back", func() {
			Expect(fs.Symlink("d2/f", "d1/link")).To(Succeed())
			Expect(EvalSymlinks(fs, "d1/d2/f1/../../../../../d1/link")).To(Equal("../../d1/d2/f"))
		})
		It("rel back symlink", func() {
			Expect(EvalSymlinks(fs, "d1/d2/f1/../..")).To(Equal("d1"))
			Expect(EvalSymlinks(fs, "d1/d2/f1/../../..")).To(Equal("."))
			Expect(EvalSymlinks(fs, "d1/d2/f1/../../../..")).To(Equal(".."))
			Expect(EvalSymlinks(fs, "d1/d2/f1/../../../../../d1")).To(Equal("../../d1"))
		})
		It("abs back", func() {
			Expect(EvalSymlinks(fs, "/d1/d2/f1/../..")).To(Equal("/d1"))
			Expect(EvalSymlinks(fs, "/d1/d2/f1/../../..")).To(Equal("/"))
			Expect(EvalSymlinks(fs, "/d1/d2/f1/../../../..")).To(Equal("/"))
		})

		It("cwd link rel back fail", func() {
			Expect(cwd.Symlink("f/..", "/d1/d2/link")).To(Succeed())
			ExpectErr(EvalSymlinks(cwd, "d2/link"))
		})
		It("cwd link rel back", func() {
			Expect(cwd.Symlink("../..", "/d1/d2/d3/link")).To(Succeed())
			Expect(EvalSymlinks(cwd, "d2/d3/link")).To(Equal("."))
			Expect(EvalSymlinks(cwd, "d2/d3/link/..")).To(Equal(".."))
			Expect(cwd.Symlink("d3/link/..", "/d1/d2/link")).To(Succeed())
			Expect(EvalSymlinks(cwd, "d2/link/..")).To(Equal("../.."))
		})
	})
})
