package git

import (
    "context"
    "fmt"
    "os"
    "path/filepath"
    "time"

    "github.com/go-git/go-git/v5"
    "github.com/go-git/go-git/v5/plumbing"
    "github.com/go-git/go-git/v5/plumbing/object"
    "github.com/schollz/progressbar/v3"
)

type ChangeStatus string

const (
    Added      ChangeStatus = "added"
    Modified   ChangeStatus = "modified"
    Deleted    ChangeStatus = "deleted"
    Renamed    ChangeStatus = "renamed"
    Unmodified ChangeStatus = "unmodified"
)

type FileChange struct {
    Path       string
    OldPath    string
    Content    string
    OldContent string
    Status     ChangeStatus
    Language   string
}

type DiffOptions struct {
    FromBranch   string
    ToBranch     string
    FromCommit   string
    ToCommit     string
    DiffMode     string
    ContextLines int
}

type CloneOptions struct {
    URL       string
    Branch    string
    CacheDir  string
    Timeout   time.Duration
    Progress  *progressbar.ProgressBar
}

type Repository struct {
    url        string
    localPath  string
    repo       *git.Repository
    isTemp     bool
    progress   *progressbar.ProgressBar
}

func NewLocalRepository(path string, progress *progressbar.ProgressBar) (*Repository, error) {
    repo, err := git.PlainOpen(path)
    if err != nil {
        return nil, fmt.Errorf("failed to open repository: %w", err)
    }

    return &Repository{
        localPath: path,
        repo:      repo,
        progress:  progress,
    }, nil
}

func NewRemoteRepository(opts CloneOptions) (*Repository, error) {
    // Create cache directory if it doesn't exist
    if err := os.MkdirAll(opts.CacheDir, 0755); err != nil {
        return nil, fmt.Errorf("failed to create cache directory: %w", err)
    }

    // Create temporary directory for clone
    tempDir, err := os.MkdirTemp(opts.CacheDir, "repo-*")
    if err != nil {
        return nil, fmt.Errorf("failed to create temporary directory: %w", err)
    }

    r := &Repository{
        url:       opts.URL,
        localPath: tempDir,
        isTemp:    true,
        progress:  opts.Progress,
    }

    // Set up clone options
    cloneOpts := &git.CloneOptions{
        URL:           opts.URL,
        Progress:      progressWriter{opts.Progress},
        SingleBranch:  true,
        Depth:         1,
    }

    if opts.Branch != "" {
        cloneOpts.ReferenceName = plumbing.NewBranchReferenceName(opts.Branch)
    }

    // Clone with timeout
    ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
    defer cancel()

    repo, err := git.PlainCloneContext(ctx, tempDir, false, cloneOpts)
    if err != nil {
        os.RemoveAll(tempDir)
        return nil, fmt.Errorf("failed to clone repository: %w", err)
    }

    r.repo = repo
    return r, nil
}

func (r *Repository) Close() error {
    if r.isTemp && r.localPath != "" {
        return os.RemoveAll(r.localPath)
    }
    return nil
}

func (r *Repository) CompareBranches(opts DiffOptions) ([]FileChange, error) {
    // Get references for both branches
    fromRef, err := r.repo.Reference(plumbing.NewBranchReferenceName(opts.FromBranch), true)
    if err != nil {
        return nil, fmt.Errorf("failed to get source branch reference: %w", err)
    }

    toRef, err := r.repo.Reference(plumbing.NewBranchReferenceName(opts.ToBranch), true)
    if err != nil {
        return nil, fmt.Errorf("failed to get target branch reference: %w", err)
    }

    // Get commit objects
    fromCommit, err := r.repo.CommitObject(fromRef.Hash())
    if err != nil {
        return nil, fmt.Errorf("failed to get source commit: %w", err)
    }

    toCommit, err := r.repo.CommitObject(toRef.Hash())
    if err != nil {
        return nil, fmt.Errorf("failed to get target commit: %w", err)
    }

    // Get patch between commits
    patch, err := fromCommit.Patch(toCommit)
    if err != nil {
        return nil, fmt.Errorf("failed to get patch: %w", err)
    }

    var changes []FileChange
    for _, filePatch := range patch.FilePatches() {
        from, to := filePatch.Files()
        change := FileChange{}

        switch {
        case from == nil && to != nil:
            // Added file
            change.Status = Added
            change.Path = to.Path()
            change.Content = getFileContent(r.repo, toCommit, to.Path())

        case from != nil && to == nil:
            // Deleted file
            change.Status = Deleted
            change.Path = from.Path()
            change.OldContent = getFileContent(r.repo, fromCommit, from.Path())

        case from != nil && to != nil:
            if from.Path() != to.Path() {
                // Renamed file
                change.Status = Renamed
                change.OldPath = from.Path()
                change.Path = to.Path()
            } else {
                // Modified file
                change.Status = Modified
                change.Path = to.Path()
            }
            change.OldContent = getFileContent(r.repo, fromCommit, from.Path())
            change.Content = getFileContent(r.repo, toCommit, to.Path())
        }

        change.Language = detectLanguage(change.Path)
        changes = append(changes, change)

        if r.progress != nil {
            r.progress.Add(1)
        }
    }

    return changes, nil
}

func getFileContent(repo *git.Repository, commit *object.Commit, path string) string {
    file, err := commit.File(path)
    if err != nil {
        return ""
    }

    content, err := file.Contents()
    if err != nil {
        return ""
    }

    return content
}

func detectLanguage(path string) string {
    ext := filepath.Ext(path)
    switch ext {
    case ".go":
        return "Go"
    case ".js":
        return "JavaScript"
    case ".py":
        return "Python"
    case ".java":
        return "Java"
    case ".cpp", ".cc", ".cxx":
        return "C++"
    case ".cs":
        return "C#"
    case ".rb":
        return "Ruby"
    case ".php":
        return "PHP"
    case ".swift":
        return "Swift"
    case ".rs":
        return "Rust"
    case ".kt":
        return "Kotlin"
    case ".ts":
        return "TypeScript"
    default:
        return "Unknown"
    }
}

type progressWriter struct {
    bar *progressbar.ProgressBar
}

func (pw progressWriter) Write(p []byte) (n int, err error) {
    if pw.bar != nil {
        pw.bar.Add(len(p))
    }
    return len(p), nil
}

func (r *Repository) GetChanges(opts DiffOptions) ([]FileChange, error) {
    if opts.FromBranch != "" && opts.ToBranch != "" {
        return r.CompareBranches(opts)
    }

    // If no branch comparison is requested, get all files from current branch
    head, err := r.repo.Head()
    if err != nil {
        return nil, fmt.Errorf("failed to get repository head: %w", err)
    }

    commit, err := r.repo.CommitObject(head.Hash())
    if err != nil {
        return nil, fmt.Errorf("failed to get commit: %w", err)
    }

    var changes []FileChange
    files, err := commit.Files()
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

    return changes, err
}
