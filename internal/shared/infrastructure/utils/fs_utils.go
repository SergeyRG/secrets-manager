package utils

import (
	"errors"
	"os"
)

func DirExists(path string) bool {
	info, err := os.Stat(path)
	if err == nil {
		return info.IsDir()
	}

	if errors.Is(err, os.ErrNotExist) {
		return false
	}

	return false
}

func FileExists(path string) bool {
	info, err := os.Stat(path)
	if err == nil {
		return !info.IsDir()
	}

	if errors.Is(err, os.ErrNotExist) {
		return false
	}

	return false
}
