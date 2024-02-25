package utils

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ParseBearer godoc
func ParseBearer(authHeader string) (string, error) {
	if strings.HasPrefix(authHeader, "Bearer ") {
		str := strings.Split(authHeader, "Bearer")
		return strings.TrimSpace(str[1]), nil
	}

	return "", errors.New("not a valid header")
}

// ParseGitHubFullName godoc
func ParseGitHubFullName(input string) (owner, name string, err error) {
	parts := strings.Split(input, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("input must be in the format 'owner/name'")
	}

	return parts[0], parts[1], nil
}

// FindFile recursively searches for a file named `filename` starting from `path`.
// It returns the full path to the file if found, or an empty string and an error if not found.
func FindFile(path, fileName string) (string, error) {
	var foundPath string
	err := filepath.Walk(path, func(currentPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Name() == fileName {
			foundPath = currentPath
			return filepath.SkipDir
		}
		return nil
	})

	if err != nil {
		return "", err
	}

	if foundPath == "" {
		return "", os.ErrNotExist
	}

	return foundPath, nil
}
