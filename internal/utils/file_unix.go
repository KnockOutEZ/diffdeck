//go:build !windows
package utils

import (
    "path/filepath"
    "strings"
)

// IsHiddenFile checks if a file is hidden on Unix-like systems
func IsHiddenFile(path string) bool {
    filename := filepath.Base(path)
    return strings.HasPrefix(filename, ".")
}