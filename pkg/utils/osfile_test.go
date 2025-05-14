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

package utils_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/mandelsoft/vfs/pkg/layerfs"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/projectionfs"
	"github.com/mandelsoft/vfs/pkg/utils"
	"github.com/mandelsoft/vfs/pkg/vfs"
)

var _ = Describe("readonly filesystem", func() {
	Context("osfs", func() {
		It("osfs", func() {

			f, err := osfs.OsFs.OpenFile("osfile.go", vfs.O_RDONLY, 0)
			Expect(err).To(Succeed())
			defer f.Close()

			Expect(utils.OSFile(f)).NotTo(BeNil())
		})
	})

	Context("projection", func() {
		It("projection", func() {

			osfs := osfs.OsFs

			p, err := vfs.Canonical(osfs, ".", true)
			Expect(err).To(Succeed())

			fs, err := projectionfs.New(osfs, p)
			Expect(err).To(Succeed())

			f, err := fs.OpenFile("osfile.go", vfs.O_RDONLY, 0)
			Expect(err).To(Succeed())
			defer f.Close()

			Expect(utils.OSFile(f)).NotTo(BeNil())
		})
	})

	Context("layer", func() {
		It("mem->os", func() {

			osfs := osfs.OsFs

			p, err := vfs.Canonical(osfs, ".", true)
			Expect(err).To(Succeed())

			pfs, err := projectionfs.New(osfs, p)
			Expect(err).To(Succeed())

			mem := memoryfs.New()
			err = vfs.WriteFile(mem, "test", []byte("test data"), 0o600)
			Expect(err).To(Succeed())

			fs := layerfs.New(mem, pfs)

			f, err := fs.OpenFile("osfile.go", vfs.O_RDONLY, 0)
			Expect(err).To(Succeed())
			defer f.Close()
			Expect(utils.OSFile(f)).NotTo(BeNil())

			f, err = fs.OpenFile("test", vfs.O_RDONLY, 0)
			Expect(err).To(Succeed())
			defer f.Close()
			Expect(utils.OSFile(f)).To(BeNil())
		})

		It("os->mem", func() {

			osfs := osfs.OsFs

			p, err := vfs.Canonical(osfs, ".", true)
			Expect(err).To(Succeed())

			pfs, err := projectionfs.New(osfs, p)
			Expect(err).To(Succeed())

			mem := memoryfs.New()
			err = vfs.WriteFile(mem, "test", []byte("test data"), 0o600)
			Expect(err).To(Succeed())

			fs := layerfs.New(pfs, mem)

			f, err := fs.OpenFile("osfile.go", vfs.O_RDONLY, 0)
			Expect(err).To(Succeed())
			defer f.Close()
			Expect(utils.OSFile(f)).NotTo(BeNil())

			f, err = fs.OpenFile("test", vfs.O_RDONLY, 0)
			Expect(err).To(Succeed())
			defer f.Close()
			Expect(utils.OSFile(f)).To(BeNil())
		})

		It("create os->mem", func() {

			osfs, err := osfs.NewTempFileSystem()
			Expect(err).To(Succeed())
			defer vfs.Cleanup(osfs)

			p, err := vfs.Canonical(osfs, ".", true)
			Expect(err).To(Succeed())

			pfs, err := projectionfs.New(osfs, p)
			Expect(err).To(Succeed())

			mem := memoryfs.New()
			err = vfs.WriteFile(mem, "test", []byte("test data"), 0o600)
			Expect(err).To(Succeed())

			fs := layerfs.New(pfs, mem)

			f, err := fs.OpenFile("osfile.go", vfs.O_CREATE, 0o600)
			Expect(err).To(Succeed())
			defer f.Close()
			Expect(utils.OSFile(f)).NotTo(BeNil())

			f, err = fs.OpenFile("test", vfs.O_RDONLY, 0)
			Expect(err).To(Succeed())
			defer f.Close()
			Expect(utils.OSFile(f)).To(BeNil())
		})
	})
})
