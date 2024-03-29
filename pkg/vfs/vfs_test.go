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

package vfs_test

import (
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/mandelsoft/vfs/pkg/cwdfs"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	. "github.com/mandelsoft/vfs/pkg/test"
	. "github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/mandelsoft/vfs/pkg/yamlfs"
)

var _ = Describe("filesystem", func() {
	var fs VFS

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

		It("anonymous", func() {
			Expect(IsAbs(nil, "/")).To(BeTrue())
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
				// fmt.Printf("trimming %+v:  %q\n", segments, path)
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
				ExpectFileCreate(fs, "f1", content, nil)
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
				ExpectRead(file, content)
				Expect(file.Close()).To(Succeed())
			})

			It("overwrite existing file", func() {
				content := []byte("Other")
				ExpectFileCreate(fs, "f1", []byte("This is a test"), nil)
				Expect(WriteFile(fs, "f1", content, os.ModePerm)).To(Succeed())

				file, err := fs.Open("f1")
				Expect(err).To(Succeed())
				ExpectRead(file, content)
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

		Context("Rel", func() {
			It("sub path", func() {
				Expect(fs.Rel("/", "/sub")).To(Equal("sub"))
				Expect(fs.Rel("/", "/sub/dir")).To(Equal("sub/dir"))
				Expect(fs.Rel("/", "/sub/dir/file")).To(Equal("sub/dir/file"))
				Expect(fs.Rel("sub", "sub/dir")).To(Equal("dir"))
				Expect(fs.Rel("sub", "sub/dir/file")).To(Equal("dir/file"))
			})

			It("parent path", func() {
				Expect(fs.Rel("/sub", "/")).To(Equal(".."))
				Expect(fs.Rel("/sub/dir", "/")).To(Equal("../.."))
				Expect(fs.Rel("/sub/dir/file", "/")).To(Equal("../../.."))
				Expect(fs.Rel("sub/dir", "sub")).To(Equal(".."))
				Expect(fs.Rel("sub/dir/file", "sub")).To(Equal("../.."))
			})

			It("common parent", func() {
				Expect(fs.Rel("/sub/dir", "/sub/file")).To(Equal("../file"))
				Expect(fs.Rel("/sub/dir/file", "/sub/dir2/file")).To(Equal("../../dir2/file"))
			})

		})
	})

	Context("eval sym links", func() {
		var fs VFS
		var cwd VFS

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
	Context("temp", func() {
		var fs VFS
		temp := "/tmp"

		BeforeEach(func() {
			fs = New(memoryfs.New())
			Expect(fs.Mkdir(temp, os.ModePerm)).To(Succeed())
		})

		AfterEach(func() {
			Cleanup(fs)
		})

		It("tempfile", func() {
			f, err := fs.TempFile(temp, "test-")
			Expect(err).To(Succeed())
			defer fs.Remove(f.Name())
			Expect(f.WriteString("tempdata")).To(Equal(8))
			Expect(f.Close()).To(Succeed())
			ExpectFileContent(fs, f.Name(), "tempdata")
		})

		It("tempdir", func() {
			path, err := fs.TempDir(temp, "test-")
			Expect(err).To(Succeed())
			defer fs.RemoveAll(path)
			d := fs.Join(path, "d1")
			Expect(fs.Mkdir(d, os.ModePerm)).To(Succeed())
			ExpectFolders(fs, path, []string{"d1"}, nil)
		})
	})

	Context("walk", func() {
		var fs *yamlfs.YamlFileSystem

		content := `
d1:
  a: This is d1a
  d2:
    a: This is file d2a
    d3:
      a: This is file d3a
    z: This is file d2z
  z: This is file d1z
`
		fs, err := yamlfs.New([]byte(content))
		if err != nil {
			Fail("invalid yaml fs: %s" + err.Error())
		}

		var order []string

		BeforeEach(func() {
			order = nil
		})

		It("walks fs", func() {
			catch := func(path string, info FileInfo, err error) error {
				order = append(order, path)
				return nil
			}
			Expect(Walk(fs, "", catch)).To(Succeed())
			Expect(order).To(Equal([]string{
				"",
				"d1",
				"d1/a",
				"d1/d2",
				"d1/d2/a",
				"d1/d2/d3",
				"d1/d2/d3/a",
				"d1/d2/z",
				"d1/z",
			}))
		})

		It("walks fs", func() {
			catch := func(path string, info FileInfo, err error) error {
				order = append(order, path)
				if path == "d1/d2/d3" {
					return SkipDir
				}
				return nil
			}
			Expect(Walk(fs, "", catch)).To(Succeed())
			Expect(order).To(Equal([]string{
				"",
				"d1",
				"d1/a",
				"d1/d2",
				"d1/d2/a",
				"d1/d2/d3",
				"d1/d2/z",
				"d1/z",
			}))
		})
	})
})
