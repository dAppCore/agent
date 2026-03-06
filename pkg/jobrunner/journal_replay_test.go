package jobrunner

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// readJournalEntries reads all JSONL entries from a given file path.
func readJournalEntries(t *testing.T, path string) []JournalEntry {
	t.Helper()
	f, err := os.Open(path)
	require.NoError(t, err)
	defer func() { _ = f.Close() }()

	var entries []JournalEntry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var entry JournalEntry
		err := json.Unmarshal(scanner.Bytes(), &entry)
		require.NoError(t, err)
		entries = append(entries, entry)
	}
	require.NoError(t, scanner.Err())
	return entries
}

// readAllJournalFiles reads all .jsonl files recursively under a base directory.
func readAllJournalFiles(t *testing.T, baseDir string) []JournalEntry {
	t.Helper()
	var all []JournalEntry
	err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if filepath.Ext(path) == ".jsonl" {
			entries := readJournalEntries(t, path)
			all = append(all, entries...)
		}
		return nil
	})
	require.NoError(t, err)
	return all
}

// --- Journal replay: write multiple entries, read back, verify round-trip ---

func TestJournal_Replay_Good_WriteAndReadBack(t *testing.T) {
	dir := t.TempDir()

	j, err := NewJournal(dir)
	require.NoError(t, err)

	baseTime := time.Date(2026, 2, 10, 10, 0, 0, 0, time.UTC)

	// Write 5 entries with different actions, times, and repos.
	entries := []struct {
		signal *PipelineSignal
		result *ActionResult
	}{
		{
			signal: &PipelineSignal{
				EpicNumber: 1, ChildNumber: 2, PRNumber: 10,
				RepoOwner: "org-a", RepoName: "repo-1",
				PRState: "OPEN", CheckStatus: "SUCCESS", Mergeable: "MERGEABLE",
			},
			result: &ActionResult{
				Action:    "enable_auto_merge",
				RepoOwner: "org-a", RepoName: "repo-1",
				Success: true, Timestamp: baseTime, Duration: 100 * time.Millisecond, Cycle: 1,
			},
		},
		{
			signal: &PipelineSignal{
				EpicNumber: 1, ChildNumber: 3, PRNumber: 11,
				RepoOwner: "org-a", RepoName: "repo-1",
				PRState: "OPEN", CheckStatus: "FAILURE", Mergeable: "CONFLICTING",
			},
			result: &ActionResult{
				Action:    "send_fix_command",
				RepoOwner: "org-a", RepoName: "repo-1",
				Success: true, Timestamp: baseTime.Add(5 * time.Minute), Duration: 50 * time.Millisecond, Cycle: 1,
			},
		},
		{
			signal: &PipelineSignal{
				EpicNumber: 5, ChildNumber: 10, PRNumber: 20,
				RepoOwner: "org-b", RepoName: "repo-2",
				PRState: "MERGED", CheckStatus: "SUCCESS", Mergeable: "UNKNOWN",
			},
			result: &ActionResult{
				Action:    "tick_parent",
				RepoOwner: "org-b", RepoName: "repo-2",
				Success: true, Timestamp: baseTime.Add(10 * time.Minute), Duration: 200 * time.Millisecond, Cycle: 2,
			},
		},
		{
			signal: &PipelineSignal{
				EpicNumber: 5, ChildNumber: 11, PRNumber: 21,
				RepoOwner: "org-b", RepoName: "repo-2",
				PRState: "OPEN", CheckStatus: "PENDING", Mergeable: "MERGEABLE",
				IsDraft: true,
			},
			result: &ActionResult{
				Action:    "publish_draft",
				RepoOwner: "org-b", RepoName: "repo-2",
				Success: false, Error: "API error", Timestamp: baseTime.Add(15 * time.Minute),
				Duration: 300 * time.Millisecond, Cycle: 2,
			},
		},
		{
			signal: &PipelineSignal{
				EpicNumber: 1, ChildNumber: 4, PRNumber: 12,
				RepoOwner: "org-a", RepoName: "repo-1",
				PRState: "OPEN", CheckStatus: "SUCCESS", Mergeable: "MERGEABLE",
				ThreadsTotal: 3, ThreadsResolved: 1,
			},
			result: &ActionResult{
				Action:    "dismiss_reviews",
				RepoOwner: "org-a", RepoName: "repo-1",
				Success: true, Timestamp: baseTime.Add(20 * time.Minute), Duration: 150 * time.Millisecond, Cycle: 3,
			},
		},
	}

	for _, e := range entries {
		err := j.Append(e.signal, e.result)
		require.NoError(t, err)
	}

	// Read back all entries.
	all := readAllJournalFiles(t, dir)
	require.Len(t, all, 5)

	// Build a map by action for flexible lookup (filepath.Walk order is by path, not insertion).
	byAction := make(map[string][]JournalEntry)
	for _, e := range all {
		byAction[e.Action] = append(byAction[e.Action], e)
	}

	// Verify enable_auto_merge entry (org-a/repo-1).
	require.Len(t, byAction["enable_auto_merge"], 1)
	eam := byAction["enable_auto_merge"][0]
	assert.Equal(t, "org-a/repo-1", eam.Repo)
	assert.Equal(t, 1, eam.Epic)
	assert.Equal(t, 2, eam.Child)
	assert.Equal(t, 10, eam.PR)
	assert.Equal(t, 1, eam.Cycle)
	assert.True(t, eam.Result.Success)
	assert.Equal(t, int64(100), eam.Result.DurationMs)

	// Verify publish_draft (failed entry has error).
	require.Len(t, byAction["publish_draft"], 1)
	pd := byAction["publish_draft"][0]
	assert.Equal(t, "publish_draft", pd.Action)
	assert.False(t, pd.Result.Success)
	assert.Equal(t, "API error", pd.Result.Error)

	// Verify signal snapshot preserves state.
	assert.True(t, pd.Signals.IsDraft)
	assert.Equal(t, "PENDING", pd.Signals.CheckStatus)

	// Verify dismiss_reviews has thread counts preserved.
	require.Len(t, byAction["dismiss_reviews"], 1)
	dr := byAction["dismiss_reviews"][0]
	assert.Equal(t, 3, dr.Signals.ThreadsTotal)
	assert.Equal(t, 1, dr.Signals.ThreadsResolved)
}

// --- Journal replay: filter by action ---

func TestJournal_Replay_Good_FilterByAction(t *testing.T) {
	dir := t.TempDir()

	j, err := NewJournal(dir)
	require.NoError(t, err)

	ts := time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC)

	actions := []string{"enable_auto_merge", "tick_parent", "send_fix_command", "tick_parent", "publish_draft"}
	for i, action := range actions {
		signal := &PipelineSignal{
			EpicNumber: 1, ChildNumber: i + 1, PRNumber: 10 + i,
			RepoOwner: "org", RepoName: "repo",
			PRState: "OPEN", CheckStatus: "SUCCESS", Mergeable: "MERGEABLE",
		}
		result := &ActionResult{
			Action:    action,
			RepoOwner: "org", RepoName: "repo",
			Success:   true,
			Timestamp: ts.Add(time.Duration(i) * time.Minute),
			Duration:  100 * time.Millisecond,
			Cycle:     i + 1,
		}
		require.NoError(t, j.Append(signal, result))
	}

	all := readAllJournalFiles(t, dir)
	require.Len(t, all, 5)

	// Filter by action=tick_parent.
	var tickParentEntries []JournalEntry
	for _, e := range all {
		if e.Action == "tick_parent" {
			tickParentEntries = append(tickParentEntries, e)
		}
	}

	assert.Len(t, tickParentEntries, 2)
	assert.Equal(t, 2, tickParentEntries[0].Child)
	assert.Equal(t, 4, tickParentEntries[1].Child)
}

// --- Journal replay: filter by repo ---

func TestJournal_Replay_Good_FilterByRepo(t *testing.T) {
	dir := t.TempDir()

	j, err := NewJournal(dir)
	require.NoError(t, err)

	ts := time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC)

	repos := []struct {
		owner string
		name  string
	}{
		{"host-uk", "core-php"},
		{"host-uk", "core-tenant"},
		{"host-uk", "core-php"},
		{"lethean", "go-scm"},
		{"host-uk", "core-tenant"},
	}

	for i, r := range repos {
		signal := &PipelineSignal{
			EpicNumber: 1, ChildNumber: i + 1, PRNumber: 10 + i,
			RepoOwner: r.owner, RepoName: r.name,
			PRState: "OPEN", CheckStatus: "SUCCESS", Mergeable: "MERGEABLE",
		}
		result := &ActionResult{
			Action:    "tick_parent",
			RepoOwner: r.owner, RepoName: r.name,
			Success:   true,
			Timestamp: ts.Add(time.Duration(i) * time.Minute),
			Duration:  50 * time.Millisecond,
			Cycle:     i + 1,
		}
		require.NoError(t, j.Append(signal, result))
	}

	// Read entries for host-uk/core-php.
	phpPath := filepath.Join(dir, "host-uk", "core-php", "2026-02-10.jsonl")
	phpEntries := readJournalEntries(t, phpPath)
	assert.Len(t, phpEntries, 2)
	for _, e := range phpEntries {
		assert.Equal(t, "host-uk/core-php", e.Repo)
	}

	// Read entries for host-uk/core-tenant.
	tenantPath := filepath.Join(dir, "host-uk", "core-tenant", "2026-02-10.jsonl")
	tenantEntries := readJournalEntries(t, tenantPath)
	assert.Len(t, tenantEntries, 2)
	for _, e := range tenantEntries {
		assert.Equal(t, "host-uk/core-tenant", e.Repo)
	}

	// Read entries for lethean/go-scm.
	scmPath := filepath.Join(dir, "lethean", "go-scm", "2026-02-10.jsonl")
	scmEntries := readJournalEntries(t, scmPath)
	assert.Len(t, scmEntries, 1)
	assert.Equal(t, "lethean/go-scm", scmEntries[0].Repo)
}

// --- Journal replay: filter by time range (date partitioning) ---

func TestJournal_Replay_Good_FilterByTimeRange(t *testing.T) {
	dir := t.TempDir()

	j, err := NewJournal(dir)
	require.NoError(t, err)

	// Write entries across three different days.
	dates := []time.Time{
		time.Date(2026, 2, 8, 9, 0, 0, 0, time.UTC),
		time.Date(2026, 2, 9, 10, 0, 0, 0, time.UTC),
		time.Date(2026, 2, 9, 14, 0, 0, 0, time.UTC),
		time.Date(2026, 2, 10, 8, 0, 0, 0, time.UTC),
		time.Date(2026, 2, 10, 16, 0, 0, 0, time.UTC),
	}

	for i, ts := range dates {
		signal := &PipelineSignal{
			EpicNumber: 1, ChildNumber: i + 1, PRNumber: 10 + i,
			RepoOwner: "org", RepoName: "repo",
			PRState: "OPEN", CheckStatus: "SUCCESS", Mergeable: "MERGEABLE",
		}
		result := &ActionResult{
			Action:    "merge",
			RepoOwner: "org", RepoName: "repo",
			Success:   true,
			Timestamp: ts,
			Duration:  100 * time.Millisecond,
			Cycle:     i + 1,
		}
		require.NoError(t, j.Append(signal, result))
	}

	// Verify each date file has the correct number of entries.
	day8Path := filepath.Join(dir, "org", "repo", "2026-02-08.jsonl")
	day8Entries := readJournalEntries(t, day8Path)
	assert.Len(t, day8Entries, 1)
	assert.Equal(t, "2026-02-08T09:00:00Z", day8Entries[0].Timestamp)

	day9Path := filepath.Join(dir, "org", "repo", "2026-02-09.jsonl")
	day9Entries := readJournalEntries(t, day9Path)
	assert.Len(t, day9Entries, 2)
	assert.Equal(t, "2026-02-09T10:00:00Z", day9Entries[0].Timestamp)
	assert.Equal(t, "2026-02-09T14:00:00Z", day9Entries[1].Timestamp)

	day10Path := filepath.Join(dir, "org", "repo", "2026-02-10.jsonl")
	day10Entries := readJournalEntries(t, day10Path)
	assert.Len(t, day10Entries, 2)

	// Simulate a time range query: get entries for Feb 9 only.
	// In a real system, you'd list files matching the date range.
	// Here we verify the date partitioning is correct.
	rangeStart := time.Date(2026, 2, 9, 0, 0, 0, 0, time.UTC)
	rangeEnd := time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC) // exclusive

	var filtered []JournalEntry
	all := readAllJournalFiles(t, dir)
	for _, e := range all {
		ts, err := time.Parse("2006-01-02T15:04:05Z", e.Timestamp)
		require.NoError(t, err)
		if !ts.Before(rangeStart) && ts.Before(rangeEnd) {
			filtered = append(filtered, e)
		}
	}

	assert.Len(t, filtered, 2)
	assert.Equal(t, 2, filtered[0].Child)
	assert.Equal(t, 3, filtered[1].Child)
}

// --- Journal replay: combined filter (action + repo + time) ---

func TestJournal_Replay_Good_CombinedFilter(t *testing.T) {
	dir := t.TempDir()

	j, err := NewJournal(dir)
	require.NoError(t, err)

	ts1 := time.Date(2026, 2, 10, 10, 0, 0, 0, time.UTC)
	ts2 := time.Date(2026, 2, 10, 11, 0, 0, 0, time.UTC)
	ts3 := time.Date(2026, 2, 11, 9, 0, 0, 0, time.UTC)

	testData := []struct {
		owner  string
		name   string
		action string
		ts     time.Time
	}{
		{"org", "repo-a", "tick_parent", ts1},
		{"org", "repo-a", "enable_auto_merge", ts1},
		{"org", "repo-b", "tick_parent", ts2},
		{"org", "repo-a", "tick_parent", ts3},
		{"org", "repo-b", "send_fix_command", ts3},
	}

	for i, td := range testData {
		signal := &PipelineSignal{
			EpicNumber: 1, ChildNumber: i + 1, PRNumber: 100 + i,
			RepoOwner: td.owner, RepoName: td.name,
			PRState: "MERGED", CheckStatus: "SUCCESS", Mergeable: "UNKNOWN",
		}
		result := &ActionResult{
			Action:    td.action,
			RepoOwner: td.owner, RepoName: td.name,
			Success:   true,
			Timestamp: td.ts,
			Duration:  50 * time.Millisecond,
			Cycle:     i + 1,
		}
		require.NoError(t, j.Append(signal, result))
	}

	// Filter: action=tick_parent AND repo=org/repo-a.
	repoAPath := filepath.Join(dir, "org", "repo-a")
	var repoAEntries []JournalEntry
	err = filepath.Walk(repoAPath, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if filepath.Ext(path) == ".jsonl" {
			entries := readJournalEntries(t, path)
			repoAEntries = append(repoAEntries, entries...)
		}
		return nil
	})
	require.NoError(t, err)

	var tickParentRepoA []JournalEntry
	for _, e := range repoAEntries {
		if e.Action == "tick_parent" && e.Repo == "org/repo-a" {
			tickParentRepoA = append(tickParentRepoA, e)
		}
	}

	assert.Len(t, tickParentRepoA, 2)
	assert.Equal(t, 1, tickParentRepoA[0].Child)
	assert.Equal(t, 4, tickParentRepoA[1].Child)
}

// --- Journal replay: empty journal returns no entries ---

func TestJournal_Replay_Good_EmptyJournal(t *testing.T) {
	dir := t.TempDir()

	all := readAllJournalFiles(t, dir)
	assert.Empty(t, all)
}

// --- Journal replay: single entry round-trip preserves all fields ---

func TestJournal_Replay_Good_FullFieldRoundTrip(t *testing.T) {
	dir := t.TempDir()

	j, err := NewJournal(dir)
	require.NoError(t, err)

	ts := time.Date(2026, 2, 15, 14, 30, 45, 0, time.UTC)

	signal := &PipelineSignal{
		EpicNumber:      42,
		ChildNumber:     7,
		PRNumber:        99,
		RepoOwner:       "host-uk",
		RepoName:        "core-admin",
		PRState:         "OPEN",
		IsDraft:         true,
		Mergeable:       "CONFLICTING",
		CheckStatus:     "FAILURE",
		ThreadsTotal:    5,
		ThreadsResolved: 2,
	}

	result := &ActionResult{
		Action:    "send_fix_command",
		RepoOwner: "host-uk",
		RepoName:  "core-admin",
		Success:   false,
		Error:     "comment API returned 503",
		Timestamp: ts,
		Duration:  1500 * time.Millisecond,
		Cycle:     7,
	}

	require.NoError(t, j.Append(signal, result))

	path := filepath.Join(dir, "host-uk", "core-admin", "2026-02-15.jsonl")
	entries := readJournalEntries(t, path)
	require.Len(t, entries, 1)

	e := entries[0]
	assert.Equal(t, "2026-02-15T14:30:45Z", e.Timestamp)
	assert.Equal(t, 42, e.Epic)
	assert.Equal(t, 7, e.Child)
	assert.Equal(t, 99, e.PR)
	assert.Equal(t, "host-uk/core-admin", e.Repo)
	assert.Equal(t, "send_fix_command", e.Action)
	assert.Equal(t, 7, e.Cycle)

	// Signal snapshot.
	assert.Equal(t, "OPEN", e.Signals.PRState)
	assert.True(t, e.Signals.IsDraft)
	assert.Equal(t, "CONFLICTING", e.Signals.Mergeable)
	assert.Equal(t, "FAILURE", e.Signals.CheckStatus)
	assert.Equal(t, 5, e.Signals.ThreadsTotal)
	assert.Equal(t, 2, e.Signals.ThreadsResolved)

	// Result snapshot.
	assert.False(t, e.Result.Success)
	assert.Equal(t, "comment API returned 503", e.Result.Error)
	assert.Equal(t, int64(1500), e.Result.DurationMs)
}

// --- Journal replay: concurrent writes produce valid JSONL ---

func TestJournal_Replay_Good_ConcurrentWrites(t *testing.T) {
	dir := t.TempDir()

	j, err := NewJournal(dir)
	require.NoError(t, err)

	ts := time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC)

	// Write 20 entries concurrently.
	done := make(chan struct{}, 20)
	for i := range 20 {
		go func(idx int) {
			signal := &PipelineSignal{
				EpicNumber: 1, ChildNumber: idx, PRNumber: idx,
				RepoOwner: "org", RepoName: "repo",
				PRState: "OPEN", CheckStatus: "SUCCESS", Mergeable: "MERGEABLE",
			}
			result := &ActionResult{
				Action:    "test",
				RepoOwner: "org", RepoName: "repo",
				Success:   true,
				Timestamp: ts,
				Duration:  10 * time.Millisecond,
				Cycle:     idx,
			}
			_ = j.Append(signal, result)
			done <- struct{}{}
		}(i)
	}

	for range 20 {
		<-done
	}

	// All entries should be parseable and present.
	path := filepath.Join(dir, "org", "repo", "2026-02-10.jsonl")
	entries := readJournalEntries(t, path)
	assert.Len(t, entries, 20)

	// Each entry should have valid JSON (no corruption from concurrent writes).
	for _, e := range entries {
		assert.NotEmpty(t, e.Action)
		assert.Equal(t, "org/repo", e.Repo)
	}
}
