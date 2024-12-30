package utils

import (
    "fmt"
    "runtime"

    "github.com/atotto/clipboard"
)

func CopyToClipboard(text string) error {
    if clipboard.Unsupported {
        return fmt.Errorf("clipboard operations not supported on %s", runtime.GOOS)
    }
    return clipboard.WriteAll(text)
}
