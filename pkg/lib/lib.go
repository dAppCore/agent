// SPDX-License-Identifier: EUPL-1.2

// Package lib provides embedded content for agent dispatch.
// Prompts, tasks, flows, personas, and workspace templates.
//
// Structure:
//
//	prompt/      — System prompts (HOW to work)
//	task/        — Structured task plans (WHAT to do)
//	task/code/   — Code-specific tasks (review, refactor, etc.)
//	flow/        — Build/release workflows per language/tool
//	persona/     — Domain/role system prompts (WHO you are)
//	workspace/   — Agent workspace templates (WHERE to work)
//
// Usage:
//
//	r := lib.Prompt("coding")        // r.Value.(string)
//	r := lib.Task("code/review")     // r.Value.(string)
//	r := lib.Persona("secops/dev")   // r.Value.(string)
//	r := lib.Flow("go")              // r.Value.(string)
//	r := lib.ExtractWorkspace("default", "/tmp/workspace", data)
//	core.Println(r.OK)
package lib

import (
	"embed"
	"sync"

	core "dappco.re/go/core"
)

//go:embed all:prompt
var promptFiles embed.FS

//go:embed all:task
var taskFiles embed.FS

//go:embed all:flow
var flowFiles embed.FS

//go:embed all:persona
var personaFiles embed.FS

//go:embed all:workspace
var workspaceFiles embed.FS

var (
	promptFS    *core.Embed
	taskFS      *core.Embed
	flowFS      *core.Embed
	personaFS   *core.Embed
	workspaceFS *core.Embed
	data        *core.Data

	mountOnce   sync.Once
	mountResult core.Result
)

// lib.MountData(c)
// r := c.Data().ReadString("prompts/coding.md")
// r := c.Data().ListNames("flows")
func MountData(c *core.Core) {
	if result := ensureMounted(); !result.OK {
		return
	}

	d := c.Data()
	d.Set("prompts", promptFS)
	d.Set("tasks", taskFS)
	d.Set("flows", flowFS)
	d.Set("personas", personaFS)
	d.Set("workspaces", workspaceFS)
}

func ensureMounted() core.Result {
	mountOnce.Do(func() {
		mountedData := &core.Data{Registry: core.NewRegistry[*core.Embed]()}

		for _, item := range []struct {
			name       string
			filesystem embed.FS
			baseDir    string
			assign     func(*core.Embed)
		}{
			{name: "prompt", filesystem: promptFiles, baseDir: "prompt", assign: func(emb *core.Embed) { promptFS = emb }},
			{name: "task", filesystem: taskFiles, baseDir: "task", assign: func(emb *core.Embed) { taskFS = emb }},
			{name: "flow", filesystem: flowFiles, baseDir: "flow", assign: func(emb *core.Embed) { flowFS = emb }},
			{name: "persona", filesystem: personaFiles, baseDir: "persona", assign: func(emb *core.Embed) { personaFS = emb }},
			{name: "workspace", filesystem: workspaceFiles, baseDir: "workspace", assign: func(emb *core.Embed) { workspaceFS = emb }},
		} {
			mounted := mountEmbed(item.filesystem, item.baseDir)
			if !mounted.OK {
				mountResult = mounted
				return
			}

			emb := mounted.Value.(*core.Embed)
			item.assign(emb)
			mountedData.Set(item.name, emb)
		}

		data = mountedData
		mountResult = core.Result{Value: mountedData, OK: true}
	})

	return mountResult
}

func mountEmbed(filesystem embed.FS, baseDir string) core.Result {
	result := core.Mount(filesystem, baseDir)
	if result.OK {
		return result
	}

	if err, ok := result.Value.(error); ok {
		return core.Result{
			Value: core.E("lib.mountEmbed", core.Concat("mount ", baseDir), err),
			OK:    false,
		}
	}

	return core.Result{
		Value: core.E("lib.mountEmbed", core.Concat("mount ", baseDir), nil),
		OK:    false,
	}
}

// --- Prompts ---

// Template tries Prompt then Task (backwards compat).
//
//	r := lib.Template("coding")
//	if r.OK { content := r.Value.(string) }
func Template(slug string) core.Result {
	if result := ensureMounted(); !result.OK {
		return result
	}
	if r := Prompt(slug); r.OK {
		return r
	}
	return Task(slug)
}

// Prompt reads a system prompt by slug.
//
//	r := lib.Prompt("coding")
//	if r.OK { content := r.Value.(string) }
func Prompt(slug string) core.Result {
	if result := ensureMounted(); !result.OK {
		return result
	}
	return promptFS.ReadString(core.Concat(slug, ".md"))
}

// Task reads a structured task plan by slug. Tries .md, .yaml, .yml.
//
//	r := lib.Task("code/review")
//	if r.OK { content := r.Value.(string) }
func Task(slug string) core.Result {
	if result := ensureMounted(); !result.OK {
		return result
	}
	for _, ext := range []string{".md", ".yaml", ".yml"} {
		if r := taskFS.ReadString(core.Concat(slug, ext)); r.OK {
			return r
		}
	}
	return core.Result{
		Value: core.E("lib.Task", core.Concat("task not found: ", slug), nil),
		OK:    false,
	}
}

// Bundle holds a task's main content plus companion files.
//
//	r := lib.TaskBundle("code/review")
//	if r.OK { b := r.Value.(lib.Bundle) }
type Bundle struct {
	Main  string
	Files map[string]string
}

// TaskBundle reads a task and its companion files.
//
//	r := lib.TaskBundle("code/review")
//	if r.OK { b := r.Value.(lib.Bundle) }
func TaskBundle(slug string) core.Result {
	if result := ensureMounted(); !result.OK {
		return result
	}
	main := Task(slug)
	if !main.OK {
		return main
	}
	b := Bundle{Main: main.Value.(string), Files: make(map[string]string)}
	r := taskFS.ReadDir(slug)
	if !r.OK {
		return core.Result{Value: b, OK: true}
	}
	nr := data.ListNames(core.JoinPath("task", slug))
	if nr.OK {
		for _, name := range nr.Value.([]string) {
			for _, ext := range []string{".md", ".yaml", ".yml", ".txt", ""} {
				fullName := core.Concat(name, ext)
				if fr := taskFS.ReadString(core.JoinPath(slug, fullName)); fr.OK {
					b.Files[fullName] = fr.Value.(string)
					break
				}
			}
		}
	}
	return core.Result{Value: b, OK: true}
}

// Flow reads a build/release workflow by slug.
//
//	r := lib.Flow("go")
//	if r.OK { content := r.Value.(string) }
func Flow(slug string) core.Result {
	if result := ensureMounted(); !result.OK {
		return result
	}
	return flowFS.ReadString(core.Concat(slug, ".md"))
}

// Persona reads a domain/role persona by path.
//
//	r := lib.Persona("secops/developer")
//	if r.OK { content := r.Value.(string) }
func Persona(path string) core.Result {
	if result := ensureMounted(); !result.OK {
		return result
	}
	return personaFS.ReadString(core.Concat(path, ".md"))
}

// --- Workspace Templates ---

// WorkspaceData is the data passed to workspace templates.
//
//	data := &lib.WorkspaceData{
//		Repo: "go-io", Task: "fix tests", Agent: "codex", BuildCmd: "go build ./...",
//	}
type WorkspaceData struct {
	Repo            string
	Branch          string
	Task            string
	Agent           string
	Language        string
	Prompt          string
	Persona         string
	Flow            string
	Context         string
	Recent          string
	Dependencies    string
	Conventions     string
	RepoDescription string
	BuildCmd        string
	TestCmd         string
}

// ExtractWorkspace creates an agent workspace from a template.
// Template names: "default", "security", "review".
//
//	r := lib.ExtractWorkspace("default", "/tmp/ws", &lib.WorkspaceData{
//	    Repo: "go-io", Task: "fix tests", Agent: "codex",
//	})
//	core.Println(r.OK)
func ExtractWorkspace(templateName, targetDir string, data *WorkspaceData) core.Result {
	if result := ensureMounted(); !result.OK {
		if err, ok := result.Value.(error); ok {
			return core.Result{
				Value: core.E("lib.ExtractWorkspace", core.Concat("mount workspace template ", templateName), err),
				OK:    false,
			}
		}
		return core.Result{
			Value: core.E("lib.ExtractWorkspace", core.Concat("mount workspace template ", templateName), nil),
			OK:    false,
		}
	}

	r := workspaceFS.Sub(templateName)
	if !r.OK {
		if err, ok := r.Value.(error); ok {
			return core.Result{
				Value: core.E("lib.ExtractWorkspace", core.Concat("template not found: ", templateName), err),
				OK:    false,
			}
		}
		return core.Result{
			Value: core.E("lib.ExtractWorkspace", core.Concat("template not found: ", templateName), nil),
			OK:    false,
		}
	}
	result := core.Extract(r.Value.(*core.Embed).FS(), targetDir, data)
	if !result.OK {
		if err, ok := result.Value.(error); ok {
			return core.Result{
				Value: core.E("lib.ExtractWorkspace", core.Concat("extract workspace template ", templateName), err),
				OK:    false,
			}
		}
		return core.Result{
			Value: core.E("lib.ExtractWorkspace", core.Concat("extract workspace template ", templateName), nil),
			OK:    false,
		}
	}
	return core.Result{Value: targetDir, OK: true}
}

// WorkspaceFile reads a single file from a workspace template.
// Returns the file content as a string.
//
//	r := lib.WorkspaceFile("default", "CODEX-PHP.md.tmpl")
//	if r.OK { content := r.Value.(string) }
func WorkspaceFile(templateName, filename string) core.Result {
	if result := ensureMounted(); !result.OK {
		return result
	}
	r := workspaceFS.Sub(templateName)
	if !r.OK {
		return r
	}
	embed := r.Value.(*core.Embed)
	return embed.ReadString(filename)
}

// --- List Functions ---

// ListPrompts returns available system prompt slugs.
//
//	prompts := lib.ListPrompts() // ["coding", "review", ...]
func ListPrompts() []string { return listNames("prompt") }

// ListFlows returns available build/release flow slugs.
//
//	flows := lib.ListFlows() // ["go", "php", "node", ...]
func ListFlows() []string { return listNames("flow") }

// ListWorkspaces returns available workspace template names.
//
//	templates := lib.ListWorkspaces() // ["default", "security", ...]
func ListWorkspaces() []string { return listNames("workspace") }

// ListTasks returns available task plan slugs, including nested paths.
//
//	tasks := lib.ListTasks() // ["bug-fix", "code/review", "code/refactor", ...]
func ListTasks() []string {
	if result := ensureMounted(); !result.OK {
		return nil
	}

	result := listNamesRecursive("task", ".")
	names := core.NewArray(result...)
	names.Deduplicate()
	return names.AsSlice()
}

// ListPersonas returns available persona paths, including nested directories.
//
//	personas := lib.ListPersonas() // ["code/go", "secops/developer", ...]
func ListPersonas() []string {
	if result := ensureMounted(); !result.OK {
		return nil
	}

	names := core.NewArray(listNamesRecursive("persona", ".")...)
	names.Deduplicate()
	return names.AsSlice()
}

// listNamesRecursive walks an embed tree via Data.ListNames.
// Directories are recursed into. Files are added as slugs (extension stripped by ListNames).
// A name can be both a file AND a directory (e.g. code/review.md + code/review/).
func listNamesRecursive(mount, dir string) []string {
	if result := ensureMounted(); !result.OK {
		return nil
	}

	path := core.JoinPath(mount, dir)
	nr := data.ListNames(path)
	if !nr.OK {
		return nil
	}

	var slugs []string
	for _, name := range nr.Value.([]string) {
		relPath := name
		if dir != "." {
			relPath = core.JoinPath(dir, name)
		}

		subPath := core.JoinPath(mount, relPath)

		// Try as directory — recurse if it has contents
		if childNames := data.ListNames(subPath); childNames.OK {
			slugs = append(slugs, listNamesRecursive(mount, relPath)...)
		}

		// Always add the slug — ListNames includes both files and dirs
		slugs = append(slugs, relPath)
	}
	return slugs
}

func listNames(mount string) []string {
	if result := ensureMounted(); !result.OK {
		return nil
	}

	r := data.ListNames(core.JoinPath(mount, "."))
	if !r.OK {
		return nil
	}
	return r.Value.([]string)
}
