//go:build windows
package utils

//go:generate mkwinsyscall -output zsyscall_windows.go file_windows.go

import (
    "golang.org/x/sys/windows"
)

// IsHiddenFile checks if a file is hidden on Windows systems
func IsHiddenFile(path string) bool {
    pointer, err := windows.UTF16PtrFromString(path)
    if err != nil {
        return false
    }
    attributes, err := windows.GetFileAttributes(pointer)
    if err != nil {
        return false
    }
    return attributes&windows.FILE_ATTRIBUTE_HIDDEN != 0
}
