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

    fi.IsSymlink = info.Mode()&os.ModeSymlink != 0

    fi.IsHidden = IsHiddenFile(path)

    fi.IsExecutable = info.Mode()&0111 != 0

    if !fi.IsDir && !fi.IsSymlink {
        content, err := os.ReadFile(path)
        if err != nil {
            return fi, nil 
        }

        mtype, isText := DetectMimeType(content)
        fi.MimeType = mtype
        fi.IsText = isText

        if fi.IsText {
            fi.Encoding, _ = DetectEncoding(content)
            fi.LineCount = CountLines(content)
        }
    }

    return fi, nil
}

func DetectMimeType(content []byte) (string, bool) {
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

func DetectEncoding(content []byte) (string, error) {
    detector := chardet.NewTextDetector()
    result, err := detector.DetectBest(content)
    if err != nil {
        return "", err
    }
    return result.Charset, nil
}

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

func CountLines(content []byte) int {
    if len(content) == 0 {
        return 0
    }

    count := 0
    for _, b := range content {
        if b == '\n' {
            count++
        }
    }

    if content[len(content)-1] != '\n' {
        count++
    }

    return count
}

