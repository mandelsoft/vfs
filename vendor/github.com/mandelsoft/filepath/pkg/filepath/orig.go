package filepath

import (
	orig "path/filepath"
)

func ToSlash(path string) string {
	return orig.ToSlash(path)
}

func FromSlash(path string) string {
	return orig.FromSlash(path)
}

func SplitList(path string) []string {
	return orig.SplitList(path)
}

func Split(path string) (string, string) {
	return orig.Split(path)
}

func Clean(path string) string {
	return orig.Clean(path)
}

func Ext(path string) string {
	return orig.Ext(path)
}

func VolumeName(path string) string {
	return orig.VolumeName(path)
}

func Rel(basepath, targpath string) (string, error) {
	base, err := Canonical(basepath, false)
	if err != nil {
		return "", err
	}
	targpath, err = EvalSymlinks(targpath)
	if err != nil {
		return "", err
	}
	return orig.Rel(base, targpath)
}

type WalkFunc orig.WalkFunc

func Walk(root string, walkFn WalkFunc) error {
	return orig.Walk(root, orig.WalkFunc(walkFn))
}
