package main

import (
    "flag"
    "fmt"
    "os"
    "path/filepath"
    "time"

    "github.com/KnockOutEZ/diffdeck/internal/config"
    "github.com/KnockOutEZ/diffdeck/internal/formatter"
    "github.com/KnockOutEZ/diffdeck/internal/git"
    "github.com/KnockOutEZ/diffdeck/internal/scanner"
    "github.com/KnockOutEZ/diffdeck/internal/security"
    "github.com/KnockOutEZ/diffdeck/internal/utils"
    "github.com/schollz/progressbar/v3"
)

var (
    version = "1.0.0"

    // Command line flags
    configPath       string
    outputPath      string
    outputStyle     string
    includePatterns string
    ignorePatterns  string
    remoteURL       string
    remoteBranch    string
    fromBranch      string
    toBranch        string
    diffMode        string
    cacheDir        string
    showVersion     bool
    initConfig      bool
    topFilesLen     int
    showLineNumbers bool
    copyToClipboard bool
    noSecurityCheck bool
    verbose         bool
    progressBar     bool
    maxFileSize     int64
    timeout         time.Duration
)

func init() {
    // Basic flags
    flag.StringVar(&configPath, "config", "", "Path to config file")
    flag.StringVar(&outputPath, "output", "", "Output file path")
    flag.StringVar(&outputStyle, "style", "plain", "Output style (plain, xml, markdown)")
    flag.StringVar(&includePatterns, "include", "", "Include patterns (comma-separated)")
    flag.StringVar(&ignorePatterns, "ignore", "", "Ignore patterns (comma-separated)")
    
    // Git-related flags
    flag.StringVar(&remoteURL, "remote", "", "Remote repository URL")
    flag.StringVar(&remoteBranch, "remote-branch", "", "Remote branch, tag, or commit")
    flag.StringVar(&fromBranch, "from-branch", "", "Source branch for comparison")
    flag.StringVar(&toBranch, "to-branch", "", "Target branch for comparison")
    flag.StringVar(&diffMode, "diff-mode", "unified", "Diff display mode (unified or side-by-side)")
    flag.StringVar(&cacheDir, "cache-dir", filepath.Join(os.TempDir(), "diffdeck-cache"), "Cache directory for remote repositories")
    
    // Output control flags
    flag.BoolVar(&showVersion, "version", false, "Show version")
    flag.BoolVar(&initConfig, "init", false, "Initialize config file")
    flag.IntVar(&topFilesLen, "top-files-len", 5, "Number of top files to display")
    flag.BoolVar(&showLineNumbers, "show-line-numbers", false, "Show line numbers")
    flag.BoolVar(&copyToClipboard, "copy", false, "Copy output to clipboard")
    flag.BoolVar(&noSecurityCheck, "no-security-check", false, "Disable security check")
    flag.BoolVar(&verbose, "verbose", false, "Enable verbose logging")
    flag.BoolVar(&progressBar, "progress", true, "Show progress bar")
    flag.Int64Var(&maxFileSize, "max-file-size", 10*1024*1024, "Maximum file size in bytes")
    flag.DurationVar(&timeout, "timeout", 5*time.Minute, "Timeout for remote operations")

    // Short versions
    flag.StringVar(&outputPath, "o", "", "Output file path (shorthand)")
    flag.StringVar(&configPath, "c", "", "Config file path (shorthand)")
    flag.StringVar(&ignorePatterns, "i", "", "Ignore patterns (shorthand)")
    flag.BoolVar(&showVersion, "v", false, "Show version (shorthand)")
}

func main() {
    startTime := time.Now()
    flag.Parse()

    if err := run(); err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }

    if verbose {
        fmt.Printf("Total execution time: %v\n", time.Since(startTime))
    }
}

func run() error {
    if showVersion {
        fmt.Printf("diffdeck version %s\n", version)
        return nil
    }

    if initConfig {
        return initializeConfig()
    }

    cfg, err := loadConfig()
    if err != nil {
        return fmt.Errorf("failed to load config: %w", err)
    }

    applyCommandLineOverrides(cfg)

    var bar *progressbar.ProgressBar
    if progressBar {
        bar = progressbar.NewOptions(-1,
            progressbar.OptionSetDescription("Processing"),
            progressbar.OptionSetItsString("files"),
            progressbar.OptionShowCount(),
            progressbar.OptionShowIts(),
            progressbar.OptionSetTheme(progressbar.Theme{
                Saucer:        "=",
                SaucerHead:    ">",
                SaucerPadding: " ",
                BarStart:      "[",
                BarEnd:        "]",
            }),
        )
    }

    var changes []git.FileChange
    if remoteURL != "" {
        changes, err = processRemoteRepository(bar)
    } else if fromBranch != "" && toBranch != "" {
        changes, err = processLocalBranchComparison(bar, cfg)
    } else {
        changes, err = processLocalFiles(cfg, bar)
    }
    if err != nil {
        return err
    }

    if !cfg.Security.DisableSecurityCheck {
        if err := runSecurityCheck(changes, bar); err != nil {
            return err
        }
    }

    output, err := formatOutput(changes, cfg)
    if err != nil {
        return err
    }

    return writeOutput(output, cfg)
}



func processRemoteRepository(bar *progressbar.ProgressBar) ([]git.FileChange, error) {
    opts := git.CloneOptions{
        URL:       remoteURL,
        Branch:    remoteBranch,
        CacheDir:  cacheDir,
        Timeout:   timeout,
        Progress:  bar,
    }

    repo, err := git.NewRemoteRepository(opts)
    if err != nil {
        return nil, fmt.Errorf("failed to create repository: %w", err)
    }
    defer repo.Close()

    return repo.GetChanges(git.DiffOptions{
        FromBranch: fromBranch,
        ToBranch:   toBranch,
        DiffMode:   diffMode,
    })
}

func processLocalBranchComparison(bar *progressbar.ProgressBar, cfg *config.Config) ([]git.FileChange, error) {
    repo, err := git.NewLocalRepository(".", bar, git.RepositoryOptions{
        IgnorePatterns: cfg.Ignore.Patterns,
        Progress:       bar,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to open local repository: %w", err)
    }
    defer repo.Close()

    return repo.CompareBranches(git.DiffOptions{
        FromBranch: fromBranch,
        ToBranch:   toBranch,
        DiffMode:   diffMode,
    })
}


func processLocalFiles(cfg *config.Config, bar *progressbar.ProgressBar) ([]git.FileChange, error) {
    paths := flag.Args()
    if len(paths) == 0 {
        paths = []string{"."}
    }

    s := scanner.NewScanner(cfg, bar)
    files, err := s.Scan(paths)
    if err != nil {
        return nil, fmt.Errorf("failed to scan files: %w", err)
    }

    var changes []git.FileChange
    for _, f := range files {
        if utils.MatchesAny(f.Path, cfg.Ignore.Patterns) {
            continue
        }
        
        changes = append(changes, git.FileChange{
            Path:    f.Path,
            Content: f.Content,
            Status:  git.Unmodified,
        })
    }

    return changes, nil
}


func runSecurityCheck(changes []git.FileChange, bar *progressbar.ProgressBar) error {
    checker := security.NewChecker(security.Options{
        MaxFileSize: maxFileSize,
        Progress:   bar,
        SkipBinaries: true,
        Severity: "WARNING",
    })

    issues, err := checker.Check(changes)
    if err != nil {
        return fmt.Errorf("security check failed: %w", err)
    }

    if len(issues) > 0 {
        fmt.Fprintln(os.Stderr, "\nSecurity Issues Found:")
        for _, issue := range issues {
            fmt.Fprintf(os.Stderr, "- %s:%d: [%s] %s\n",
                issue.FilePath,
                issue.Line,
                issue.Rule,
                issue.Description)
        }
        fmt.Fprintln(os.Stderr)
    }

    return nil
}

func formatOutput(changes []git.FileChange, cfg *config.Config) (string, error) {
    f := formatter.NewFormatter(formatter.Options{
        Style:          cfg.Output.Style,
        ShowLineNumbers: cfg.Output.ShowLineNumbers,
        TopFilesLength: cfg.Output.TopFilesLength,
        DiffMode:      diffMode,
    })

    return f.Format(changes)
}

func writeOutput(output string, cfg *config.Config) error {
    if cfg.Output.FilePath != "" {
        if err := os.WriteFile(cfg.Output.FilePath, []byte(output), 0644); err != nil {
            return fmt.Errorf("failed to write output file: %w", err)
        }
    }

    if cfg.Output.CopyToClipboard {
        if err := utils.CopyToClipboard(output); err != nil {
            return fmt.Errorf("failed to copy to clipboard: %w", err)
        }
    }

    if cfg.Output.FilePath == "" {
        fmt.Print(output)
    }

    return nil
}

func loadConfig() (*config.Config, error) {
    if configPath == "" {
        configPath = "diffdeck.config.json"
    }
    return config.Load(configPath)
}

func initializeConfig() error {
    cfg := config.DefaultConfig()
    return cfg.Save("diffdeck.config.json")
}

func applyCommandLineOverrides(cfg *config.Config) {
    if outputPath != "" {
        cfg.Output.FilePath = outputPath
    }
    if outputStyle != "" {
        cfg.Output.Style = outputStyle
    }
    if includePatterns != "" {
        cfg.Include = utils.ParsePatternList(includePatterns)
    }
    if ignorePatterns != "" {
        cfg.Ignore.Patterns = utils.ParsePatternList(ignorePatterns)
    }
    if showLineNumbers {
        cfg.Output.ShowLineNumbers = true
    }
    if copyToClipboard {
        cfg.Output.CopyToClipboard = true
    }
    if noSecurityCheck {
        cfg.Security.DisableSecurityCheck = true
    }
    if topFilesLen > 0 {
        cfg.Output.TopFilesLength = topFilesLen
    }
}

func logVerbose(format string, args ...interface{}) {
    if verbose {
        fmt.Fprintf(os.Stderr, format+"\n", args...)
    }
}