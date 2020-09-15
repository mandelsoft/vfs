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

package yamlfs

import (
	"errors"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/mandelsoft/vfs/pkg/test"
	"github.com/mandelsoft/vfs/pkg/vfs"
)

var _ = Describe("memory filesystem", func() {

	Context("local", func() {
		var fs *YamlFileSystem

		content := `
d1:
  d2:
    f1: This is a test
    f2:
      $type: file
      value: |
        ---Start Binary---
        VGhpcyBpcyBhIGRlY29kZWQgdGVzdAo=
        ---End Binary---
    f3: !!binary |
        VGhpcyBpcyBhIGRlY29kZWQgdGVzdAo=
    yaml:
      $type: yaml
      value: 
        map:
          field: value
        list:
          - a
          - b
    json:
      $type: json
      value:
        map:
          field: value
        list:
          - a
          - b
`
		BeforeEach(func() {
			var err error
			fs, err = New([]byte(content))
			if err != nil {
				panic(err)
			}
		})

		It("root dir", func() {
			ExpectFolders(fs, "/", []string{"d1"}, nil)
		})
		It("other dirs", func() {
			ExpectFolders(fs, "/d1", []string{"d2"}, nil)
			ExpectFolders(fs, "/d1/d2", []string{"f1", "f2", "f3", "json", "yaml"}, nil)
		})
		It("read", func() {
			ExpectFileContent(fs, "/d1/d2/f1", "This is a test")
			ExpectFileContent(fs, "/d1/d2/f2", "This is a decoded test\n")
		})
		It("yaml", func() {
			ExpectFileContent(fs, "/d1/d2/yaml", "list:\n- a\n- b\nmap:\n  field: value\n")
		})
		It("json", func() {
			ExpectFileContent(fs, "/d1/d2/json", "{\"list\":[\"a\",\"b\"],\"map\":{\"field\":\"value\"}}")
		})

		It("write json", func() {
			ExpectFileWrite(fs, "/d1/d2/json", os.O_TRUNC, "{\"list\":[\"c\",\"d\"]}", true)
			d, err := fs.Data()
			Expect(string(d)).To(Equal(`d1:
  d2:
    f1: This is a test
    f2:
      $type: file
      value: |
        ---Start Binary---
        VGhpcyBpcyBhIGRlY29kZWQgdGVzdAo=
        ---End Binary---
    f3: |
      This is a decoded test
    json:
      $type: json
      value:
        list:
        - c
        - d
    yaml:
      $type: yaml
      value:
        list:
        - a
        - b
        map:
          field: value
`))
			Expect(err).To(Succeed())
		})

		It("write binary", func() {
			ExpectFileWrite(fs, "/d1/d2/f3", os.O_TRUNC, []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 255}, true)
			d, err := fs.Data()
			Expect(err).To(Succeed())
			Expect(string(d)).To(Equal(`d1:
  d2:
    f1: This is a test
    f2:
      $type: file
      value: |
        ---Start Binary---
        VGhpcyBpcyBhIGRlY29kZWQgdGVzdAo=
        ---End Binary---
    f3: !!binary AQIDBAUGBwgJCv8=
    json:
      $type: json
      value:
        list:
        - a
        - b
        map:
          field: value
    yaml:
      $type: yaml
      value:
        list:
        - a
        - b
        map:
          field: value
`))
		})

	})

	Context("standard", func() {
		StandardTest(func() vfs.FileSystem { return NewByData(nil) })
	})

	Context("rename", func() {
		var fs *YamlFileSystem

		BeforeEach(func() {
			fs = NewByData(nil)
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
			ExpectFolders(fs, "d1/", []string{"d1n2"}, nil)
			s, err := fs.Data()
			Expect(err).To(Succeed())
			Expect(string(s)).To(Equal("d1:\n  d1n2: {}\nd2:\n  d1n1a: {}\n"))
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
