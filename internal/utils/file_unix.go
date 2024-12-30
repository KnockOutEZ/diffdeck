//go:build !windows
package utils

import (
    "path/filepath"
    "strings"
)

func IsHiddenFile(path string) bool {
    filename := filepath.Base(path)
    return strings.HasPrefix(filename, ".")
}