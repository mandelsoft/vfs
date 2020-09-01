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

package main

import (
	"fmt"
	"os"

	"github.com/mandelsoft/vfs/pkg/compose"
	"github.com/mandelsoft/vfs/pkg/cwd"
	"github.com/mandelsoft/vfs/pkg/memory"
	osfs "github.com/mandelsoft/vfs/pkg/os"
	"github.com/mandelsoft/vfs/pkg/projection"
	"github.com/mandelsoft/vfs/pkg/vfs"
)

func Error(err error) {
	fmt.Fprintf(os.Stderr, "%s\n", err)
	os.Exit(1)
}

func Assert(err error) {
	if err != nil {
		Error(err)
	}

}
func list(fs vfs.FileSystem, path string) {
	f, err := fs.Open(path)
	Assert(err)
	list, err := f.Readdirnames(0)
	Assert(err)
	f.Close()

	fmt.Printf("*** %s: %s ***\n", fs.Name(), path)
	for _, e := range list {
		fmt.Printf("%s\n", e)
	}
}

func main() {
	fs := osfs.New()

	cur, err := os.Getwd()
	Assert(err)

	list(fs, cur)

	me, err := projection.New(fs, cur)
	Assert(err)
	list(me, "/")

	cwdfs, err := cwd.New(fs, cur)
	Assert(err)
	list(cwdfs, "test")

	wdfs, err := projection.New(cwdfs, "test")
	Assert(err)
	list(wdfs, ".")
	list(wdfs, "sub")

	cfs := compose.New(wdfs)
	err = cfs.Mount("sub/m", me)
	Assert(err)
	list(cfs, "sub/m/pkg")
	list(cfs, "sub/m/pkg/../..")

	mem := memory.New()
	err = mem.Mkdir("test", os.ModePerm)
	Assert(err)
	err = mem.MkdirAll("a/b/c", os.ModePerm)
	Assert(err)
	list(mem, "/")
	list(mem, "/a")

	f, err := mem.Create("/a/file")
	Assert(err)
	_, err = f.WriteString("This is a test\n")
	Assert(err)
	f.Close()
	f, err = mem.Open("/a/file")
	Assert(err)
	buf := [10]byte{}
	n, err := f.Read(buf[:])
	Assert(err)
	if n != 10 && string(buf[:n]) != "This is a " {
		Error(fmt.Errorf("read 1 failed: %d %q", n, string(buf[:n])))
	}
	n, err = f.Read(buf[:])
	if n != 5 && string(buf[:n]) != "test\n" {
		Error(fmt.Errorf("read 2 failed: %d %q", n, string(buf[:n])))
	}
	f.Close()
	list(wdfs, "d")
	list(wdfs, "pkg")

}
