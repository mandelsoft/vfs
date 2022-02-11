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

package main

import (
	"fmt"
	"reflect"

	"gopkg.in/yaml.v2"
)

func Error(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	in := `
data: !!binary |
  VGhpcyBpcyBhIGRlY29kZWQgdGVzdAo=
`

	s := map[interface{}]interface{}{}
	err := yaml.Unmarshal([]byte(in), s)
	Error(err)

	fmt.Printf("%d\n", len(s))
	fmt.Printf("DATA: %s: %s\n", reflect.TypeOf(s["data"]), s["data"])

	s["data"] = string([]byte{255, 255})
	b, err := yaml.Marshal(s)
	Error(err)
	fmt.Printf("RESULT:\n%s\n", b)
}
