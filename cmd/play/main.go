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
	"reflect"
)

func main() {
	fi, err := os.Open("/lib/cpp2")
	if err != nil {
		fmt.Printf("open: %s(%s)\n", reflect.TypeOf(err), err)
		os.Exit(1)
	}
	names, err := fi.Readdirnames(0)
	if err != nil {
		fmt.Printf("dir: %s(%s)\n", reflect.TypeOf(err), err)
		os.Exit(1)
	}
	for _, n := range names {
		fmt.Printf("%s\n", n)
	}
}
