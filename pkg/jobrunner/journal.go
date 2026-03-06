package jobrunner

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

// validPathComponent matches safe repo owner/name characters (alphanumeric, hyphen, underscore, dot).
var validPathComponent = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)

// JournalEntry is a single line in the JSONL audit log.
type JournalEntry struct {
	Timestamp string         `json:"ts"`
	Epic      int            `json:"epic"`
	Child     int            `json:"child"`
	PR        int            `json:"pr"`
	Repo      string         `json:"repo"`
	Action    string         `json:"action"`
	Signals   SignalSnapshot `json:"signals"`
	Result    ResultSnapshot `json:"result"`
	Cycle     int            `json:"cycle"`
}

// SignalSnapshot captures the structural state of a PR at the time of action.
type SignalSnapshot struct {
	PRState         string `json:"pr_state"`
	IsDraft         bool   `json:"is_draft"`
	CheckStatus     string `json:"check_status"`
	Mergeable       string `json:"mergeable"`
	ThreadsTotal    int    `json:"threads_total"`
	ThreadsResolved int    `json:"threads_resolved"`
}

// ResultSnapshot captures the outcome of an action.
type ResultSnapshot struct {
	Success    bool   `json:"success"`
	Error      string `json:"error,omitempty"`
	DurationMs int64  `json:"duration_ms"`
}

// Journal writes ActionResult entries to date-partitioned JSONL files.
type Journal struct {
	baseDir string
	mu      sync.Mutex
}

// NewJournal creates a new Journal rooted at baseDir.
func NewJournal(baseDir string) (*Journal, error) {
	if baseDir == "" {
		return nil, errors.New("journal.NewJournal: base directory is required")
	}
	return &Journal{baseDir: baseDir}, nil
}

// sanitizePathComponent validates a single path component (owner or repo name)
// to prevent path traversal attacks. It rejects "..", empty strings, paths
// containing separators, and any value outside the safe character set.
func sanitizePathComponent(name string) (string, error) {
	// Reject empty or whitespace-only values.
	if name == "" || strings.TrimSpace(name) == "" {
		return "", fmt.Errorf("journal.sanitizePathComponent: invalid path component: %q", name)
	}

	// Reject inputs containing path separators (directory traversal attempt).
	if strings.ContainsAny(name, `/\`) {
		return "", fmt.Errorf("journal.sanitizePathComponent: path component contains directory separator: %q", name)
	}

	// Use filepath.Clean to normalize (e.g., collapse redundant dots).
	clean := filepath.Clean(name)

	// Reject traversal components.
	if clean == "." || clean == ".." {
		return "", fmt.Errorf("journal.sanitizePathComponent: invalid path component: %q", name)
	}

	// Validate against the safe character set.
	if !validPathComponent.MatchString(clean) {
		return "", fmt.Errorf("journal.sanitizePathComponent: path component contains invalid characters: %q", name)
	}

	return clean, nil
}

// ReadEntries returns an iterator over JournalEntry lines in a date-partitioned file.
func (j *Journal) ReadEntries(path string) iter.Seq2[JournalEntry, error] {
	return func(yield func(JournalEntry, error) bool) {
		f, err := os.Open(path)
		if err != nil {
			yield(JournalEntry{}, err)
			return
		}
		defer func() { _ = f.Close() }()

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			var entry JournalEntry
			if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
				if !yield(JournalEntry{}, err) {
					return
				}
				continue
			}
			if !yield(entry, nil) {
				return
			}
		}
		if err := scanner.Err(); err != nil {
			yield(JournalEntry{}, err)
		}
	}
}

// Append writes a journal entry for the given signal and result.
func (j *Journal) Append(signal *PipelineSignal, result *ActionResult) error {
	if signal == nil {
		return errors.New("journal.Append: signal is required")
	}
	if result == nil {
		return errors.New("journal.Append: result is required")
	}

	entry := JournalEntry{
		Timestamp: result.Timestamp.UTC().Format("2006-01-02T15:04:05Z"),
		Epic:      signal.EpicNumber,
		Child:     signal.ChildNumber,
		PR:        signal.PRNumber,
		Repo:      signal.RepoFullName(),
		Action:    result.Action,
		Signals: SignalSnapshot{
			PRState:         signal.PRState,
			IsDraft:         signal.IsDraft,
			CheckStatus:     signal.CheckStatus,
			Mergeable:       signal.Mergeable,
			ThreadsTotal:    signal.ThreadsTotal,
			ThreadsResolved: signal.ThreadsResolved,
		},
		Result: ResultSnapshot{
			Success:    result.Success,
			Error:      result.Error,
			DurationMs: result.Duration.Milliseconds(),
		},
		Cycle: result.Cycle,
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("journal.Append: marshal entry: %w", err)
	}
	data = append(data, '\n')

	// Sanitize path components to prevent path traversal (CVE: issue #46).
	owner, err := sanitizePathComponent(signal.RepoOwner)
	if err != nil {
		return fmt.Errorf("journal.Append: invalid repo owner: %w", err)
	}
	repo, err := sanitizePathComponent(signal.RepoName)
	if err != nil {
		return fmt.Errorf("journal.Append: invalid repo name: %w", err)
	}

	date := result.Timestamp.UTC().Format("2006-01-02")
	dir := filepath.Join(j.baseDir, owner, repo)

	// Resolve to absolute path and verify it stays within baseDir.
	absBase, err := filepath.Abs(j.baseDir)
	if err != nil {
		return fmt.Errorf("journal.Append: resolve base directory: %w", err)
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("journal.Append: resolve journal directory: %w", err)
	}
	if !strings.HasPrefix(absDir, absBase+string(filepath.Separator)) {
		return fmt.Errorf("journal.Append: path %q escapes base directory %q", absDir, absBase)
	}

	j.mu.Lock()
	defer j.mu.Unlock()

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("journal.Append: create directory: %w", err)
	}

	path := filepath.Join(dir, date+".jsonl")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("journal.Append: open file: %w", err)
	}
	defer func() { _ = f.Close() }()

	_, err = f.Write(data)
	return err
}
