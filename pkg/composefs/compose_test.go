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

package composefs_test

import (
	"os"

	"github.com/mandelsoft/vfs/pkg/composefs"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/mandelsoft/vfs/pkg/test"
	"github.com/mandelsoft/vfs/pkg/vfs"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func New() vfs.FileSystem {
	return composefs.New(memoryfs.New())
}

var _ = Describe("compose filesystem", func() {

	test.StandardTest(New)

	Context("mkdir", func() {
		var fs *composefs.ComposedFileSystem

		BeforeEach(func() {
			mem := memoryfs.New()
			fs = composefs.New(memoryfs.New())
			Expect(fs.Mkdir("/tmp", 0770)).To(Succeed())
			test.ExpectFolders(fs, "/", []string{"tmp"}, nil)
			Expect(fs.Mount("/tmp", mem)).To(Succeed())
		})

		It("partial mkdirall", func() {
			Expect(fs.MkdirAll("/tmp/d1/d2", os.ModePerm)).To(Succeed())
			Expect(fs.MkdirAll("/tmp/d1/d2/d3/d4", os.ModePerm)).To(Succeed())
			test.ExpectFolders(fs, "/tmp", []string{"d1"}, nil)
			test.ExpectFolders(fs, "/tmp/d1", []string{"d2"}, nil)
			test.ExpectFolders(fs, "/tmp/d1/d2", []string{"d3"}, nil)
			test.ExpectFolders(fs, "/tmp/d1/d2/d3", []string{"d4"}, nil)
			test.ExpectFolders(fs, "/tmp/d1/d2/d3/d4", nil, nil)
		})
	})
})
