package utils

import (
    "fmt"
    "runtime"

    "github.com/atotto/clipboard"
)

// CopyToClipboard copies text to the system clipboard
func CopyToClipboard(text string) error {
    if !clipboard.Unsupported {
        return clipboard.WriteAll(text)
    }
    return fmt.Errorf("clipboard operations not supported on %s", runtime.GOOS)
}

// ReadFromClipboard reads text from the system clipboard
func ReadFromClipboard() (string, error) {
    if !clipboard.Unsupported {
        return clipboard.ReadAll()
    }
    return "", fmt.Errorf("clipboard operations not supported on %s", runtime.GOOS)
}
