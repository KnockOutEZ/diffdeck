package utils

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/saintfish/chardet"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/unicode"
)

// FileInfo represents detailed information about a file
type FileInfo struct {
    Path          string
    Size          int64
    ModTime       int64
    IsDir         bool
    IsSymlink     bool
    IsHidden      bool
    MimeType      string
    Encoding      string
    LineCount     int
    IsText        bool
    IsExecutable  bool
}

// GetFileInfo returns detailed information about a file
func GetFileInfo(path string) (*FileInfo, error) {
    info, err := os.Lstat(path)
    if err != nil {
        return nil, err
    }

    fi := &FileInfo{
        Path:    path,
        Size:    info.Size(),
        ModTime: info.ModTime().Unix(),
        IsDir:   info.IsDir(),
    }

    // Check if it's a symlink
    fi.IsSymlink = info.Mode()&os.ModeSymlink != 0

    // Check if it's hidden
    fi.IsHidden = IsHiddenFile(path)

    // Check if it's executable
    fi.IsExecutable = info.Mode()&0111 != 0

    // Get MIME type and text status
    if !fi.IsDir && !fi.IsSymlink {
        content, err := os.ReadFile(path)
        if err != nil {
            return fi, nil // Return what we have if we can't read the file
        }

        mtype, isText := DetectMimeType(content)
        fi.MimeType = mtype
        fi.IsText = isText

        if fi.IsText {
            // Detect encoding
            fi.Encoding, _ = DetectEncoding(content)
            // Count lines
            fi.LineCount = CountLines(content)
        }
    }

    return fi, nil
}

// DetectMimeType detects the MIME type of content
func DetectMimeType(content []byte) (string, bool) {
    // Read first 512 bytes for MIME detection
    buffer := content
    if len(buffer) > 512 {
        buffer = buffer[:512]
    }

    mtype := http.DetectContentType(buffer)
    isText := strings.HasPrefix(mtype, "text/") ||
        mtype == "application/json" ||
        mtype == "application/xml" ||
        mtype == "application/javascript"

    return mtype, isText
}

// DetectEncoding detects the character encoding of content
func DetectEncoding(content []byte) (string, error) {
    detector := chardet.NewTextDetector()
    result, err := detector.DetectBest(content)
    if err != nil {
        return "", err
    }
    return result.Charset, nil
}

// ReadFileWithEncoding reads a file with the specified encoding
func ReadFileWithEncoding(path string, encodingName string) (string, error) {
    content, err := os.ReadFile(path)
    if err != nil {
        return "", err
    }

    var decoder *encoding.Decoder
    switch strings.ToLower(encodingName) {
    case "utf-8", "utf8":
        return string(content), nil
    case "utf-16le":
        decoder = unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewDecoder()
    case "utf-16be":
        decoder = unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM).NewDecoder()
    default:
        return "", fmt.Errorf("unsupported encoding: %s", encodingName)
    }

    decoded, err := decoder.Bytes(content)
    if err != nil {
        return "", err
    }

    return string(decoded), nil
}

// CountLines counts the number of lines in content
func CountLines(content []byte) int {
    if len(content) == 0 {
        return 0
    }

    // Count newlines
    count := 0
    for _, b := range content {
        if b == '\n' {
            count++
        }
    }

    // Add one if file doesn't end with newline
    if content[len(content)-1] != '\n' {
        count++
    }

    return count
}

