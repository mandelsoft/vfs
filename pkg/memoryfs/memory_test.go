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

package memoryfs

import (
	"errors"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/mandelsoft/vfs/pkg/test"
	. "github.com/mandelsoft/vfs/pkg/test"
	"github.com/mandelsoft/vfs/pkg/vfs"
)

var _ = Describe("memory filesystem", func() {
	var fs vfs.FileSystem

	BeforeEach(func() {
		fs = New()
	})

	test.StandardTest(New)

	Context("rename", func() {
		BeforeEach(func() {
			fs.MkdirAll("d1/d1n1/d1n1a", os.ModePerm)
			fs.MkdirAll("d1/d1n2", os.ModePerm)
		})
		It("rename top level", func() {
			Expect(fs.Rename("/d1", "d2")).To(Succeed())
			ExpectFolders(fs, "d2/", []string{"d1n1", "d1n2"}, nil)
		})
		It("rename sub level", func() {
			Expect(fs.Rename("/d1/d1n1", "d2")).To(Succeed())
			ExpectFolders(fs, "d2/", []string{"d1n1a"}, nil)
		})
		It("fail rename root", func() {
			Expect(fs.Rename("/", "d2")).To(Equal(errors.New("cannot rename root dir")))
		})
		It("fail rename to existent", func() {
			Expect(fs.Rename("/d1/d1n1", "d1/d1n2")).To(Equal(os.ErrExist))
		})

		It("rename link", func() {
			Expect(fs.MkdirAll("d2", os.ModePerm)).To(Succeed())
			Expect(fs.Symlink("/d1/d1n1", "d2/link")).To(Succeed())
			Expect(fs.Rename("d2/link", "d2/new")).To(Succeed())
			ExpectFolders(fs, "d2", []string{"new"}, nil)
			ExpectFolders(fs, "d2/new", []string{"d1n1a"}, nil)
		})
	})
})
