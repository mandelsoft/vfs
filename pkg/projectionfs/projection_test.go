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

package projectionfs_test

import (
	"os"

	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/projectionfs"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/mandelsoft/vfs/pkg/memoryfs"
	. "github.com/mandelsoft/vfs/pkg/test"
	"github.com/mandelsoft/vfs/pkg/vfs"
)

var _ = Describe("projection filesystem", func() {
	Context("plain", func() {
		var fs vfs.FileSystem
		var mem vfs.FileSystem

		BeforeEach(func() {
			var err error

			mem = memoryfs.New()

			mem.MkdirAll("d1/d1d1/d1d1d1/a", os.ModePerm)
			mem.MkdirAll("d1/d1d1/d1d1d2/b", os.ModePerm)
			mem.MkdirAll("d2/d2d1", os.ModePerm)
			fs, err = projectionfs.New(mem, "d1")
			Expect(err).To(Succeed())
		})

		It("root", func() {
			ExpectFolders(fs, "/", []string{"d1d1"}, nil)
			ExpectFolders(fs, "/d1d1", []string{"d1d1d1", "d1d1d2"}, nil)
		})

		It("visibility", func() {
			ExpectFolders(fs, "..", []string{"d1d1"}, nil)
			ExpectFolders(fs, "d1d1/..", []string{"d1d1"}, nil)
			ExpectFolders(fs, "d1d1/../..", []string{"d1d1"}, nil)
		})

		It("abolute symlink", func() {
			fs.Symlink("/d1d1/d1d1d1", "d1d1/link")
			ExpectFolders(fs, "d1d1/link", []string{"a"}, nil)
		})
		It("relative symlink", func() {
			fs.Symlink("./d1d1d1", "d1d1/link")
			ExpectFolders(fs, "d1d1/link", []string{"a"}, nil)
		})
		It("remove symlink", func() {
			fs.Symlink("./d1d1d1", "d1d1/link")
			ExpectFolders(fs, "d1d1", []string{"d1d1d1", "d1d1d2", "link"}, nil)
			Expect(fs.Remove("/d1d1/link")).To(Succeed())
			ExpectFolders(fs, "d1d1", []string{"d1d1d1", "d1d1d2"}, nil)
		})
		It("symlink visibility", func() {
			fs.Symlink("../../..", "d1d1/link")
			ExpectFolders(fs, "d1d1/link", []string{"d1d1"}, nil)
		})
		It("stat symlink", func() {
			fs.Symlink("/d1d1/d1d1d1", "d1d1/link")
			fi, err := fs.Stat("d1d1/link")
			Expect(err).To(Succeed())
			Expect(fi.IsDir()).To(BeTrue())
		})
		It("lstat symlink", func() {
			fs.Symlink("/d1d1/d1d1d1", "d1d1/link")
			fi, err := fs.Lstat("d1d1/link")
			Expect(err).To(Succeed())
			Expect(fi.Mode() & (os.ModeType)).To(Equal(os.ModeSymlink))
		})

		It("stat non existing", func() {
			_, err := fs.Stat("/d1d1/none/none")
			Expect(vfs.IsNotExist(err)).To(BeTrue())
			Expect(vfs.DirExists(fs, "/d1d1/none/none")).To(BeFalse())
		})

		It("stat non existing2", func() {
			Expect(vfs.DirExists(osfs.OsFs, "/bar/baz")).To(BeFalse())

			proFs, _ := projectionfs.New(osfs.OsFs, "/")
			Expect(vfs.DirExists(proFs, "/bar/baz")).To(BeFalse())
		})

		It("stat non existing2", func() {
			Expect(vfs.Exists(osfs.OsFs, "/bar/baz")).To(BeFalse())

			proFs, _ := projectionfs.New(osfs.OsFs, "/")
			Expect(vfs.Exists(proFs, "/bar/baz")).To(BeFalse())
		})

		It("provides single root", func() {
			Expect(projectionfs.Root(fs)).To(Equal("/d1"))
		})
		It("provides nested root", func() {
			nested, err := projectionfs.New(fs, "/d1d1")
			Expect(err).To(Succeed())
			Expect(projectionfs.Root(nested)).To(Equal("/d1/d1d1"))
		})
	})

	Context("tempfs", func() {
		var tempfs vfs.FileSystem

		BeforeEach(func() {
			t, err := osfs.NewTempFileSystem()
			Expect(err).To(Succeed())
			tempfs = t
		})

		AfterEach(func() {
			vfs.Cleanup(tempfs)
		})

		It("handles permission problem", func() {
			Expect(tempfs.Mkdir("test", 0)).To(Succeed())
			err := tempfs.Mkdir("test/sub", 0)
			Expect(err).To(HaveOccurred())
			Expect(os.IsPermission(err)).To(BeTrue())
		})
	})
})
