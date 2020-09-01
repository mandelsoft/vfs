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

package vfs

import (
	"errors"
	"os"
	"path"
	"strings"

	"github.com/mandelsoft/filepath/pkg/filepath"
)

// IsPathSeparator reports whether c is a directory separator character.
func IsPathSeparator(c uint8) bool {
	return PathSeparatorChar == c
}

// Join joins any number of path elements into a single path, adding
// a Separator if necessary. Join never calls Clean on the result to
// assure the result denotes the same file as the input.
// On Windows, the result is a UNC path if and only if the first path
// element is a UNC path.
func Join(fs FileSystem, elem ...string) string {
	r := Trim(fs, strings.Join(elem, PathSeparatorString))
	vol, p := SplitVolume(fs, r)
	for strings.Index(p, PathSeparatorString+PathSeparatorString) >= 0 {
		p = strings.ReplaceAll(p, PathSeparatorString+PathSeparatorString, PathSeparatorString)
	}
	return vol + p
}

// Clean returns the shortest path name equivalent to path
// by purely lexical processing. It applies the following rules
// iteratively until no further processing can be done:
//
//	1. Replace multiple path separators with a single one.
//	2. Eliminate each . path name element (the current directory).
//	3. Eliminate each inner .. path name element (the parent directory)
//	   along with the non-.. element that precedes it.
//	4. Eliminate .. elements that begin a rooted path:
//	   that is, replace "/.." by "/" at the beginning of a path.
//
// The returned path ends in a slash only if it is the root "/".
//
// If the result of this process is an empty string, Clean
// returns the string ".".
func Clean(fs FileSystem, p string) string {
	p = fs.Normalize(p)
	vol := fs.VolumeName(p)
	return vol + path.Clean(p[len(vol):])
}

// Dir returns the path's directory dropping the final element
// after removing trailing Separators, Dir does not call Clean on the path.
// If the path is empty, Dir returns "." or "/" for a rooted path.
// If the path consists entirely of Separators, Dir2 returns a single Separator.
// The returned path does not end in a Separator unless it is the root directory.
// This function is the counterpart of Base
// Base("a/b/")="b" and Dir("a/b/") = "a".
func Dir(fs FileSystem, path string) string {
	def := "."
	vol, path := SplitVolume(fs, path)
	i := len(path) - 1
	for i > 0 && IsPathSeparator(path[i]) {
		i--
	}
	for i >= 0 && !IsPathSeparator(path[i]) {
		i--
	}
	for i > 0 && IsPathSeparator(path[i]) {
		def = PathSeparatorString
		i--
	}
	path = path[0 : i+1]
	if path == "" {
		path = def
	}
	return vol + path
}

func Base(fs FileSystem, path string) string {
	_, path = SplitVolume(fs, path)
	i := len(path) - 1
	for i > 0 && IsPathSeparator(path[i]) {
		i--
	}
	j := i
	for j >= 0 && !IsPathSeparator(path[j]) {
		j--
	}
	path = path[j+1 : i+1]
	if path == "" {
		if j == 0 {
			return PathSeparatorString
		}
		return "."
	}
	return path
}

// Trim eleminates trailing slashes from a path name.
// An empty path is unchanged.
func Trim(fs FileSystem, path string) string {
	path = fs.Normalize(path)
	vol := fs.VolumeName(path)
	i := len(path) - 1
	for i > len(vol) && IsPathSeparator(path[i]) {
		i--
	}
	p := path[:i+1]

	return p
}

// IsAbs return true if the given path is an absolute one
// starting with a Separator or is quailified by a volume name.
func IsAbs(fs FileSystem, path string) bool {
	_, path = SplitVolume(fs, path)
	return strings.HasPrefix(path, PathSeparatorString)
}

// IsRoot determines whether a given path is a root path.
// This might be the separator or the separator preceded by
// a volume name.
func IsRoot(fs FileSystem, path string) bool {
	_, path = SplitVolume(fs, path)
	return path == PathSeparatorString
}

func SplitVolume(fs FileSystem, path string) (string, string) {
	path = fs.Normalize(path)
	vol := fs.VolumeName(path)
	return vol, path[len(vol):]
}

// Canonical returns the canonical absolute path of a file.
// If exist=false the denoted file must not exist, but
// then the part of the initial path refering to a not existing
// directory structure is lexically resolved (like Clean) and
// does not consider potential symbolic links that might occur
// if the file is finally created in the future.
func Canonical(fs FileSystem, path string, exist bool) (string, error) {
	return walk(fs, path, -1, exist)
}

// EvalSymLinks resolves all symbolic links in a path
// and returns a path not containing any symbolic link
// anymore. It does not call Clean on a non-canonical path,
// so the result always denotes the same file than the original path.
// If the given path is a relative one, a
// reLative one is returned as long as there is no
// absolute symbolic link and the relative path does
// not goes up the current working diretory.
// If a relative path is returned, symbolic links
// up the current working directory are not resolved.
func EvalSymlinks(fs FileSystem, path string) (string, error) {
	return walk(fs, path, 0, false)
}

// Abs returns an absolute representation of path.
// If the path is not absolute it will be joined with the current
// working directory to turn it into an absolute path. The absolute
// path name for a given file is not guaranteed to be unique.
// Symbolic links in the given path will be resolved, but not in
// the current working directory, if used to make the path absolute.
// The denoted file may not exist.
// Abs never calls Clean on the result, so the resulting path
// will denote the same file as the argument.
func Abs(fs FileSystem, path string) (string, error) {
	path, err := walk(fs, path, 0, false)
	if err != nil {
		return "", err
	}
	if IsAbs(fs, path) {
		return path, nil
	}
	p, err := fs.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Join(p, path), nil
}

func walk(fs FileSystem, p string, parent int, exist bool) (string, error) {
	p = fs.Normalize(p)
	var rest []string = []string{}

	links := 0

	for !IsRoot(fs, p) && p != "" {
		n, b := Split(fs, p)
		if b == "" {
			p = n
			continue
		}
		fi, err := fs.Lstat(p)
		if Exists_(err) {
			if err != nil && !os.IsPermission(err) {
				return "", err
			}
			if fi.Mode()&os.ModeSymlink != 0 {
				newpath, err := fs.Readlink(p)
				if err != nil {
					return "", err
				}
				newpath = fs.Normalize(newpath)
				if IsAbs(fs, newpath) {
					p = newpath
				} else {
					p = filepath.Join(n, newpath)
				}
				links++
				if links > 255 {
					return "", errors.New("AbsPath: too many links")
				}
				continue
			}
		} else {
			if exist {
				return "", err
			}
		}
		if b != "." {
			rest = append([]string{b}, rest...)
			if parent >= 0 && b == ".." {
				parent++
			} else {
				if parent > 0 {
					parent--
				}
			}
		}
		if parent != 0 && n == "" {
			p, err = fs.Getwd()
			if err != nil {
				return "", err
			}
		} else {
			p = n
		}
	}
	if p == "" {
		return filepath.Clean(filepath.Join(rest...)), nil
	}
	return filepath.Clean(filepath.Join(append([]string{p}, rest...)...)), nil
}

func Exists_(err error) bool {
	return err == nil || !os.IsNotExist(err)
}

// Exists checks whether a file exists.
func Exists(fs FileSystem, path string) bool {
	_, err := fs.Stat(path)
	return Exists_(err)
}

// Split splits path immediately following the final Separator,
// separating it into a directory and file name component.
// If there is no Separator in path, Split returns an empty dir
// and file set to path. In contrast to filepath.Split the directory
// path does not end with a trailing Separator, so Split can
// subsequently called for the directory part, again.
func Split(fs FileSystem, path string) (dir, file string) {
	path = fs.Normalize(path)
	vol := fs.VolumeName(path)
	i := len(path) - 1
	for i >= len(vol) && !IsPathSeparator(path[i]) {
		i--
	}
	j := i
	for j > len(vol) && IsPathSeparator(path[j]) {
		j--
	}
	return path[:j+1], path[i+1:]
}

// SplitPath splits a path into a volume and an array of the path segments
func SplitPath(fs FileSystem, path string) (string, []string, bool) {
	vol, path := SplitVolume(fs, path)
	elems := []string{}
	for path != "" {
		i := 0
		for i < len(path) && IsPathSeparator(path[i]) {
			i++
		}
		j := i
		for j < len(path) && !IsPathSeparator(path[j]) {
			j++
		}
		b := path[i:j]
		path = path[j:]
		if b == "." || b == "" {
			continue
		}
		elems = append(elems, b)
	}
	return vol, elems, strings.HasPrefix(path, PathSeparatorString)
}
