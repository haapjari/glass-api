package utils

import (
	"errors"
	"fmt"
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
