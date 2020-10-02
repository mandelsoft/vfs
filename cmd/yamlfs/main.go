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
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"

	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/mandelsoft/vfs/pkg/yamlfs"
)

type Config struct {
	Go        bool
	Name      string
	Pkg       string
	Copyright string
}

func ExitOnError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}

func main() {
	var opts Config
	flag.BoolVar(&opts.Go, "go", false, "generate a go file instead of a yaml file")
	flag.StringVar(&opts.Name, "name", "YamlFS", "The go variable name to use")
	flag.StringVar(&opts.Pkg, "package", "gofs", "Package name to use")
	flag.StringVar(&opts.Copyright, "copyright", "", "Package copyright header")
	flag.Parse()
	args := flag.Args()

	srcDir := "."
	dstDir := vfs.PathSeparatorString
	if len(args) > 0 {
		if len(args) > 1 {
			if len(args) > 2 {
				ExitOnError(fmt.Errorf("too many arguments"))
			}
			dstDir = args[1]
		}
		srcDir = args[0]
	}
	dst, err := yamlfs.New(nil)
	ExitOnError(err)
	src := osfs.New()
	ExitOnError(vfs.CopyDir(src, srcDir, dst, dstDir))
	data, err := dst.Data()
	ExitOnError(err)
	if opts.Go {
		copyright := ""
		if opts.Copyright != "" {
			data, err := ioutil.ReadFile(opts.Copyright)
			ExitOnError(err)
			copyright = string(data) + "\n"
		}
		fmt.Printf(`%spackage %s

import (
	"github.com/mandelsoft/vfs/pkg/yamlfs"
)

func New%s() *yamlfs.YamlFileSystem {
	fs, _ := yamlfs.New([]byte(%s))
	return fs
}

`, copyright, opts.Pkg, opts.Name, strconv.Quote(string(data)))
	} else {
		fmt.Printf("%s\n", data)
	}
}
