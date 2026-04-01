// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"sort"

	core "dappco.re/go/core"
)

const brainSeedMemoryDefaultAgent = "virgil"

const brainSeedMemoryDefaultPath = "~/.claude/projects/*/memory/"

type BrainSeedMemoryInput struct {
	WorkspaceID int
	AgentID     string
	Path        string
	DryRun      bool
}

type BrainSeedMemoryOutput struct {
	Success     bool   `json:"success"`
	WorkspaceID int    `json:"workspace_id,omitempty"`
	AgentID     string `json:"agent_id,omitempty"`
	Path        string `json:"path,omitempty"`
	Files       int    `json:"files,omitempty"`
	Imported    int    `json:"imported,omitempty"`
	Skipped     int    `json:"skipped,omitempty"`
	DryRun      bool   `json:"dry_run,omitempty"`
}

type brainSeedMemorySection struct {
	Heading string
	Content string
}

// result := c.Command("brain/seed-memory").Run(ctx, core.NewOptions(
//
//	core.Option{Key: "workspace", Value: "1"},
//	core.Option{Key: "path", Value: "/Users/snider/.claude/projects/*/memory/"},
//
// ))
func (s *PrepSubsystem) cmdBrainSeedMemory(options core.Options) core.Result {
	return s.cmdBrainSeedMemoryLike(options, "brain seed-memory", "agentic.cmdBrainSeedMemory")
}

// result := c.Command("brain/ingest").Run(ctx, core.NewOptions(
//
//	core.Option{Key: "workspace", Value: "1"},
//	core.Option{Key: "path", Value: "/Users/snider/.claude/projects/*/memory/"},
//
// ))
func (s *PrepSubsystem) cmdBrainIngest(options core.Options) core.Result {
	return s.cmdBrainSeedMemoryLike(options, "brain ingest", "agentic.cmdBrainIngest")
}

func (s *PrepSubsystem) cmdBrainSeedMemoryLike(options core.Options, commandName string, errorLabel string) core.Result {
	input := BrainSeedMemoryInput{
		WorkspaceID: parseIntString(optionStringValue(options, "workspace", "workspace_id", "workspace-id", "_arg")),
		AgentID:     optionStringValue(options, "agent", "agent_id", "agent-id"),
		Path:        optionStringValue(options, "path"),
		DryRun:      optionBoolValue(options, "dry-run"),
	}
	if input.WorkspaceID == 0 {
		core.Print(nil, "usage: core-agent %s --workspace=1 [--agent=virgil] [--path=~/.claude/projects/*/memory/] [--dry-run]", commandName)
		return core.Result{Value: core.E(errorLabel, "workspace is required", nil), OK: false}
	}
	if input.AgentID == "" {
		input.AgentID = brainSeedMemoryDefaultAgent
	}
	if input.Path == "" {
		input.Path = brainSeedMemoryDefaultPath
	}

	result := s.brainSeedMemory(s.commandContext(), input)
	if !result.OK {
		err := commandResultError(errorLabel, result)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	output, ok := result.Value.(BrainSeedMemoryOutput)
	if !ok {
		err := core.E(errorLabel, "invalid brain seed memory output", nil)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	if output.Files == 0 {
		core.Print(nil, "No markdown files found in: %s", output.Path)
		return core.Result{Value: output, OK: true}
	}

	prefix := ""
	if output.DryRun {
		prefix = "[DRY RUN] "
	}
	core.Print(nil, "%sImported %d memories, skipped %d.", prefix, output.Imported, output.Skipped)
	return core.Result{Value: output, OK: true}
}

func (s *PrepSubsystem) brainSeedMemory(ctx context.Context, input BrainSeedMemoryInput) core.Result {
	if s.brainKey == "" {
		return core.Result{Value: core.E("agentic.brainSeedMemory", "no brain API key configured", nil), OK: false}
	}

	scanPath := brainSeedMemoryScanPath(input.Path)
	files := brainSeedMemoryFiles(scanPath)
	output := BrainSeedMemoryOutput{
		Success:     true,
		WorkspaceID: input.WorkspaceID,
		AgentID:     input.AgentID,
		Path:        scanPath,
		Files:       len(files),
		DryRun:      input.DryRun,
	}

	if len(files) == 0 {
		return core.Result{Value: output, OK: true}
	}

	for _, path := range files {
		readResult := fs.Read(path)
		if !readResult.OK {
			output.Skipped++
			continue
		}

		sections := brainSeedMemorySections(readResult.Value.(string))
		if len(sections) == 0 {
			output.Skipped++
			core.Print(nil, "  Skipped %s (no sections found)", core.PathBase(path))
			continue
		}

		project := brainSeedMemoryProject(path)
		filename := core.TrimSuffix(core.PathBase(path), ".md")

		for _, section := range sections {
			memoryType := brainSeedMemoryType(section.Heading, section.Content)
			if input.DryRun {
				core.Print(nil, "  [DRY RUN] %s :: %s (%s) — %d chars", core.PathBase(path), section.Heading, memoryType, core.RuneCount(section.Content))
				output.Imported++
				continue
			}

			body := map[string]any{
				"workspace_id": input.WorkspaceID,
				"agent_id":     input.AgentID,
				"type":         memoryType,
				"content":      core.Concat(section.Heading, "\n\n", section.Content),
				"tags":         brainSeedMemoryTags(filename),
				"project":      project,
				"confidence":   0.7,
			}

			if result := HTTPPost(ctx, core.Concat(s.brainURL, "/v1/brain/remember"), core.JSONMarshalString(body), s.brainKey, "Bearer"); !result.OK {
				output.Skipped++
				core.Print(nil, "  Failed to import %s :: %s", core.PathBase(path), section.Heading)
				continue
			}

			output.Imported++
		}
	}

	return core.Result{Value: output, OK: true}
}

func brainSeedMemoryScanPath(path string) string {
	trimmed := brainSeedMemoryExpandHome(core.Trim(path))
	if trimmed == "" {
		return brainSeedMemoryExpandHome(brainSeedMemoryDefaultPath)
	}
	if fs.IsFile(trimmed) || core.HasSuffix(trimmed, ".md") {
		return trimmed
	}
	return core.JoinPath(trimmed, "*.md")
}

func brainSeedMemoryExpandHome(path string) string {
	if core.HasPrefix(path, "~/") {
		return core.Concat(HomeDir(), core.TrimPrefix(path, "~"))
	}
	return path
}

func brainSeedMemoryFiles(scanPath string) []string {
	if scanPath == "" {
		return nil
	}
	if fs.IsFile(scanPath) {
		return []string{scanPath}
	}

	files := core.PathGlob(scanPath)
	sort.Strings(files)
	return files
}

func brainSeedMemorySections(content string) []brainSeedMemorySection {
	lines := core.Split(content, "\n")
	var sections []brainSeedMemorySection

	currentHeading := ""
	var currentContent []string

	flush := func() {
		if currentHeading == "" || len(currentContent) == 0 {
			return
		}
		sectionContent := core.Trim(core.Join("\n", currentContent...))
		if sectionContent == "" {
			return
		}
		sections = append(sections, brainSeedMemorySection{
			Heading: currentHeading,
			Content: sectionContent,
		})
	}

	for _, line := range lines {
		if heading, ok := brainSeedMemoryHeading(line); ok {
			flush()
			currentHeading = heading
			currentContent = currentContent[:0]
			continue
		}
		if currentHeading == "" {
			continue
		}
		currentContent = append(currentContent, line)
	}

	flush()
	return sections
}

func brainSeedMemoryHeading(line string) (string, bool) {
	trimmed := core.Trim(line)
	if trimmed == "" || !core.HasPrefix(trimmed, "#") {
		return "", false
	}

	hashes := 0
	for _, r := range trimmed {
		if r != '#' {
			break
		}
		hashes++
	}

	if hashes < 1 || hashes > 3 || len(trimmed) <= hashes || trimmed[hashes] != ' ' {
		return "", false
	}

	heading := core.Trim(trimmed[hashes:])
	if heading == "" {
		return "", false
	}
	return heading, true
}

func brainSeedMemoryType(heading, content string) string {
	lower := core.Lower(core.Concat(heading, " ", content))
	for _, candidate := range []struct {
		memoryType string
		keywords   []string
	}{
		{memoryType: "architecture", keywords: []string{"architecture", "stack", "infrastructure", "layer", "service mesh"}},
		{memoryType: "convention", keywords: []string{"convention", "standard", "naming", "pattern", "rule", "coding"}},
		{memoryType: "decision", keywords: []string{"decision", "chose", "strategy", "approach", "domain"}},
		{memoryType: "bug", keywords: []string{"bug", "fix", "broken", "error", "issue", "lesson"}},
		{memoryType: "plan", keywords: []string{"plan", "todo", "roadmap", "milestone", "phase"}},
		{memoryType: "research", keywords: []string{"research", "finding", "discovery", "analysis", "rfc"}},
	} {
		for _, keyword := range candidate.keywords {
			if core.Contains(lower, keyword) {
				return candidate.memoryType
			}
		}
	}
	return "observation"
}

func brainSeedMemoryTags(filename string) []string {
	if filename == "" {
		return []string{"memory-import"}
	}

	tags := []string{}
	if core.Lower(filename) != "memory" {
		tag := core.Replace(core.Replace(filename, "-", " "), "_", " ")
		if tag != "" {
			tags = append(tags, tag)
		}
	}
	tags = append(tags, "memory-import")
	return tags
}

func brainSeedMemoryProject(path string) string {
	normalised := core.Replace(path, "\\", "/")
	segments := core.Split(normalised, "/")
	for i := 1; i < len(segments); i++ {
		if segments[i] != "memory" {
			continue
		}
		projectSegment := segments[i-1]
		if projectSegment == "" {
			continue
		}
		chunks := core.Split(projectSegment, "-")
		for j := len(chunks) - 1; j >= 0; j-- {
			if chunks[j] != "" {
				return chunks[j]
			}
		}
	}
	return ""
}
