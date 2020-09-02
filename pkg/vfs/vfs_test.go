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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/mandelsoft/vfs/pkg/memoryfs"
	. "github.com/mandelsoft/vfs/pkg/vfs"
)

var _ = Describe("filesystem", func() {
	var fs *VFS

	BeforeEach(func() {
		fs = New(memoryfs.New())
	})

	Context("utils", func() {
		It("trim", func() {
			Expect(fs.Trim("path/")).To(Equal("path"))
			Expect(fs.Trim("path//other")).To(Equal("path/other"))
			Expect(fs.Trim("/path/other")).To(Equal("/path/other"))
			Expect(fs.Trim("path/other")).To(Equal("path/other"))
			Expect(fs.Trim("//path//other")).To(Equal("/path/other"))
			Expect(fs.Trim("//")).To(Equal("/"))
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

	})
})
