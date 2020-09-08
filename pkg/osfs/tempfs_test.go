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

package osfs

import (
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/mandelsoft/vfs/pkg/test"
	"github.com/mandelsoft/vfs/pkg/vfs"
)

var _ = Describe("temp", func() {
	var fs vfs.VFS
	temp := ""

	BeforeEach(func() {
		t, err := NewTempFileSystem()
		Expect(err).To(Succeed())
		fs = vfs.New(t)
	})

	AfterEach(func() {
		vfs.Cleanup(fs)
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
