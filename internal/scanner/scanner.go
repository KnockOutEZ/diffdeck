package scanner

import (
    "fmt"
    "io/fs"
    "os"
    "path/filepath"
    "strings"
    "sync"

    "github.com/KnockOutEZ/diffdeck/internal/config"
    "github.com/bmatcuk/doublestar/v4"
    "github.com/schollz/progressbar/v3"
)

type File struct {
    Path     string
    Content  string
    Size     int64
    IsDir    bool
    Children []File
}

type Scanner struct {
    progress  *progressbar.ProgressBar
    config    *config.Config
    maxSize   int64
    mu        sync.Mutex
    wg        sync.WaitGroup
    semaphore chan struct{}
}

func NewScanner(cfg *config.Config, progress *progressbar.ProgressBar) *Scanner {
    return &Scanner{
        config:    cfg,
        progress:  progress,
        maxSize:   cfg.Security.MaxFileSize,
        semaphore: make(chan struct{}, 10),
    }
}

func (s *Scanner) Scan(paths []string) ([]File, error) {
    var files []File
    for _, path := range paths {
        stat, err := os.Stat(path)
        if err != nil {
            return nil, fmt.Errorf("failed to stat %s: %w", path, err)
        }

        if stat.IsDir() {
            dirFiles, err := s.scanDirectory(path)
            if err != nil {
                return nil, err
            }
            files = append(files, dirFiles...)
        } else {
            if !s.shouldIgnore(path) {
                file, err := s.scanFile(path)
                if err != nil {
                    return nil, err
                }
                files = append(files, file)
            }
        }
    }

    s.wg.Wait()
    return files, nil
}

func (s *Scanner) scanDirectory(root string) ([]File, error) {
    var files []File
    var mu sync.Mutex

    err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
        if err != nil {
            return err
        }

        relPath, err := filepath.Rel(root, path)
        if err != nil {
            return err
        }

        if relPath == "." {
            return nil
        }

        if s.shouldIgnore(relPath) {
            if d.IsDir() {
                return filepath.SkipDir
            }
            return nil
        }

        if !s.shouldInclude(relPath) {
            if d.IsDir() {
                return filepath.SkipDir
            }
            return nil
        }

        if !d.IsDir() {
            s.wg.Add(1)
            go func() {
                defer s.wg.Done()
                s.semaphore <- struct{}{}
                defer func() { <-s.semaphore }()

                file, err := s.scanFile(path)
                if err != nil {
                    fmt.Fprintf(os.Stderr, "Error scanning %s: %v\n", path, err)
                    return
                }

                mu.Lock()
                files = append(files, file)
                mu.Unlock()

                if s.progress != nil {
                    s.progress.Add(1)
                }
            }()
        }

        return nil
    })

    return files, err
}

func (s *Scanner) shouldIgnore(path string) bool {
    path = filepath.ToSlash(path)

    for _, pattern := range s.config.Ignore.Patterns {
        pattern = filepath.ToSlash(pattern)

        if strings.HasPrefix(pattern, "**/") {
            if matched, _ := doublestar.Match(pattern, path); matched {
                return true
            }
        } else if strings.Contains(pattern, "**") {
            if matched, _ := doublestar.Match(pattern, path); matched {
                return true
            }
        } else {
            if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
                return true
            }
        }
    }

    return false
}

func (s *Scanner) shouldInclude(path string) bool {
    if len(s.config.Include) == 0 {
        return true
    }

    path = filepath.ToSlash(path)
    for _, pattern := range s.config.Include {
        pattern = filepath.ToSlash(pattern)
        if matched, _ := doublestar.Match(pattern, path); matched {
            return true
        }
    }

    return false
}

func (s *Scanner) scanFile(path string) (File, error) {
    info, err := os.Stat(path)
    if err != nil {
        return File{}, err
    }

    file := File{
        Path:  path,
        IsDir: info.IsDir(),
        Size:  info.Size(),
    }

    if !file.IsDir && file.Size <= s.maxSize {
        content, err := os.ReadFile(path)
        if err != nil {
            return file, err
        }
        file.Content = string(content)
    }

    return file, nil
}