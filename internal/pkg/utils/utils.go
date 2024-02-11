package utils

import (
	"errors"
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
