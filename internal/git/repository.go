package git

import (
    "errors"
    "fmt"
    "io"
    "net/url"
    "os"
    "path/filepath"
    "strings"

    "github.com/go-git/go-git/v5"
    "github.com/go-git/go-git/v5/plumbing"
    "github.com/go-git/go-git/v5/plumbing/object"
)

// Repository represents a Git repository
type Repository struct {
    URL           string
    Branch        string
    LocalPath     string
    repo          *git.Repository
    isTemporary   bool
}

// CloneOptions represents options for cloning a repository
type CloneOptions struct {
    Branch      string // Branch, tag, or commit hash to clone
    Depth       int    // Depth for shallow clone (0 for full clone)
    Progress    io.Writer // Writer for progress information
}

// DiffOptions represents options for generating diffs
type DiffOptions struct {
    IgnoreWhitespace bool
    ContextLines     int
    FromCommit       string
    ToCommit         string
}

// FileChange represents a changed file in the repository
type FileChange struct {
    Path     string
    Content  string
    Status   ChangeStatus
    OldPath  string // For renamed files
    Language string // Detected programming language
}

// ChangeStatus represents the type of change
type ChangeStatus string

const (
    Added      ChangeStatus = "added"
    Modified   ChangeStatus = "modified"
    Deleted    ChangeStatus = "deleted"
    Renamed    ChangeStatus = "renamed"
    Unmodified ChangeStatus = "unmodified"
)

// New creates a new Repository instance
func New(repoURL string, opts CloneOptions) (*Repository, error) {
    // Handle GitHub shorthand (e.g., "username/repo")
    if !strings.Contains(repoURL, "://") && strings.Count(repoURL, "/") == 1 {
        repoURL = "https://github.com/" + repoURL + ".git"
    }

    // Validate URL
    if _, err := url.Parse(repoURL); err != nil {
        return nil, fmt.Errorf("invalid repository URL: %w", err)
    }

    // Create temporary directory for cloning
    tempDir, err := os.MkdirTemp("", "diffdeck-*")
    if err != nil {
        return nil, fmt.Errorf("failed to create temporary directory: %w", err)
    }

    return &Repository{
        URL:         repoURL,
        Branch:      opts.Branch,
        LocalPath:   tempDir,
        isTemporary: true,
    }, nil
}

// Clone clones the repository
func (r *Repository) Clone(opts CloneOptions) error {
    // Prepare clone options
    cloneOpts := &git.CloneOptions{
        URL:           r.URL,
        Progress:      opts.Progress,
        SingleBranch:  true,
        Tags:          git.NoTags,
    }

    if opts.Depth > 0 {
        cloneOpts.Depth = opts.Depth
    }

    if opts.Branch != "" {
        cloneOpts.ReferenceName = plumbing.NewBranchReferenceName(opts.Branch)
    }

    // Clone the repository
    repo, err := git.PlainClone(r.LocalPath, false, cloneOpts)
    if err != nil {
        return fmt.Errorf("failed to clone repository: %w", err)
    }

    r.repo = repo
    return nil
}

// GetChanges returns the changes between two commits
func (r *Repository) GetChanges(opts DiffOptions) ([]FileChange, error) {
    if r.repo == nil {
        return nil, errors.New("repository not cloned")
    }

    // Get the repository head
    head, err := r.repo.Head()
    if err != nil {
        return nil, fmt.Errorf("failed to get repository head: %w", err)
    }

    // Get the commit objects
    var fromCommit, toCommit *object.Commit
    
    if opts.FromCommit != "" {
        fromHash := plumbing.NewHash(opts.FromCommit)
        fromCommit, err = r.repo.CommitObject(fromHash)
        if err != nil {
            return nil, fmt.Errorf("failed to get 'from' commit: %w", err)
        }
    }

    if opts.ToCommit != "" {
        toHash := plumbing.NewHash(opts.ToCommit)
        toCommit, err = r.repo.CommitObject(toHash)
    } else {
        toCommit, err = r.repo.CommitObject(head.Hash())
    }
    if err != nil {
        return nil, fmt.Errorf("failed to get 'to' commit: %w", err)
    }

    // Get the changes between commits
    changes := make([]FileChange, 0)
    
    if fromCommit != nil {
        patch, err := fromCommit.Patch(toCommit)
        if err != nil {
            return nil, fmt.Errorf("failed to get patch: %w", err)
        }

        for _, filePatch := range patch.FilePatches() {
            from, to := filePatch.Files()
            change := FileChange{}

            switch {
            case from == nil && to != nil:
                // Added file
                change.Status = Added
                change.Path = to.Path()
                content, err := getFileContent(r.repo, toCommit, to.Path())
                if err != nil {
                    return nil, err
                }
                change.Content = content

            case from != nil && to == nil:
                // Deleted file
                change.Status = Deleted
                change.Path = from.Path()
                content, err := getFileContent(r.repo, fromCommit, from.Path())
                if err != nil {
                    return nil, err
                }
                change.Content = content

            case from != nil && to != nil && from.Path() != to.Path():
                // Renamed file
                change.Status = Renamed
                change.OldPath = from.Path()
                change.Path = to.Path()
                content, err := getFileContent(r.repo, toCommit, to.Path())
                if err != nil {
                    return nil, err
                }
                change.Content = content

            default:
                // Modified file
                change.Status = Modified
                change.Path = to.Path()
                content, err := getFileContent(r.repo, toCommit, to.Path())
                if err != nil {
                    return nil, err
                }
                change.Content = content
            }

            change.Language = detectLanguage(change.Path)
            changes = append(changes, change)
        }
    } else {
        // If no fromCommit specified, include all files in current commit
        files, err := toCommit.Files()
        if err != nil {
            return nil, fmt.Errorf("failed to get files: %w", err)
        }

        err = files.ForEach(func(f *object.File) error {
            content, err := f.Contents()
            if err != nil {
                return err
            }

            changes = append(changes, FileChange{
                Path:     f.Name,
                Content:  content,
                Status:   Unmodified,
                Language: detectLanguage(f.Name),
            })
            return nil
        })
        if err != nil {
            return nil, fmt.Errorf("failed to process files: %w", err)
        }
    }

    return changes, nil
}

// Close cleans up repository resources
func (r *Repository) Close() error {
    if r.isTemporary && r.LocalPath != "" {
        if err := os.RemoveAll(r.LocalPath); err != nil {
            return fmt.Errorf("failed to remove temporary directory: %w", err)
        }
    }
    return nil
}

// Helper functions

func getFileContent(repo *git.Repository, commit *object.Commit, path string) (string, error) {
    file, err := commit.File(path)
    if err != nil {
        return "", fmt.Errorf("failed to get file %s: %w", path, err)
    }

    content, err := file.Contents()
    if err != nil {
        return "", fmt.Errorf("failed to get contents of %s: %w", path, err)
    }

    return content, nil
}

func detectLanguage(path string) string {
    ext := strings.ToLower(filepath.Ext(path))
    switch ext {
    case ".go":
        return "Go"
    case ".js":
        return "JavaScript"
    case ".ts":
        return "TypeScript"
    case ".py":
        return "Python"
    case ".java":
        return "Java"
    case ".rb":
        return "Ruby"
    case ".php":
        return "PHP"
    case ".cs":
        return "C#"
    case ".cpp", ".cc":
        return "C++"
    case ".h", ".hpp":
        return "C++"
    case ".rs":
        return "Rust"
    case ".swift":
        return "Swift"
    case ".kt":
        return "Kotlin"
    case ".scala":
        return "Scala"
    case ".m":
        return "Objective-C"
    case ".mm":
        return "Objective-C++"
    case ".pl":
        return "Perl"
    case ".sh":
        return "Shell"
    case ".html":
        return "HTML"
    case ".css":
        return "CSS"
    case ".scss":
        return "SCSS"
    case ".sass":
        return "Sass"
    case ".less":
        return "Less"
    case ".json":
        return "JSON"
    case ".xml":
        return "XML"
    case ".yaml", ".yml":
        return "YAML"
    case ".md":
        return "Markdown"
    case ".sql":
        return "SQL"
    default:
        return "Unknown"
    }
}
