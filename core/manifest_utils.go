package core

import (
	"path/filepath"
	"strings"

	"github.com/mitchellh/go-homedir"
)

func normalizePathForComparison(path string) (string, error) {
	expandedPath, err := homedir.Expand(path)
	if err != nil {
		return "", err
	}

	absPath, err := filepath.Abs(expandedPath)
	if err != nil {
		return "", err
	}

	cleanPath := filepath.Clean(absPath)

	cleanPath = strings.TrimPrefix(cleanPath, "/private")

	return cleanPath, nil
}
