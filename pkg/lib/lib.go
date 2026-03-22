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
//	lib.ExtractWorkspace("default", "/tmp/ws", data)
package lib

import (
	"embed"
	"io/fs"
	"path/filepath"

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
	promptFS    = mustMount(promptFiles, "prompt")
	taskFS      = mustMount(taskFiles, "task")
	flowFS      = mustMount(flowFiles, "flow")
	personaFS   = mustMount(personaFiles, "persona")
	workspaceFS = mustMount(workspaceFiles, "workspace")
)

func mustMount(fsys embed.FS, basedir string) *core.Embed {
	r := core.Mount(fsys, basedir)
	if !r.OK {
		panic(r.Value)
	}
	return r.Value.(*core.Embed)
}

// --- Prompts ---

// Template tries Prompt then Task (backwards compat).
//
//	r := lib.Template("coding")
//	if r.OK { content := r.Value.(string) }
func Template(slug string) core.Result {
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
	return promptFS.ReadString(slug + ".md")
}

// Task reads a structured task plan by slug. Tries .md, .yaml, .yml.
//
//	r := lib.Task("code/review")
//	if r.OK { content := r.Value.(string) }
func Task(slug string) core.Result {
	for _, ext := range []string{".md", ".yaml", ".yml"} {
		if r := taskFS.ReadString(slug + ext); r.OK {
			return r
		}
	}
	return core.Result{Value: fs.ErrNotExist}
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
	main := Task(slug)
	if !main.OK {
		return main
	}
	b := Bundle{Main: main.Value.(string), Files: make(map[string]string)}
	r := taskFS.ReadDir(slug)
	if !r.OK {
		return core.Result{Value: b, OK: true}
	}
	for _, e := range r.Value.([]fs.DirEntry) {
		if e.IsDir() {
			continue
		}
		if fr := taskFS.ReadString(slug + "/" + e.Name()); fr.OK {
			b.Files[e.Name()] = fr.Value.(string)
		}
	}
	return core.Result{Value: b, OK: true}
}

// Flow reads a build/release workflow by slug.
//
//	r := lib.Flow("go")
//	if r.OK { content := r.Value.(string) }
func Flow(slug string) core.Result {
	return flowFS.ReadString(slug + ".md")
}

// Persona reads a domain/role persona by path.
//
//	r := lib.Persona("secops/developer")
//	if r.OK { content := r.Value.(string) }
func Persona(path string) core.Result {
	return personaFS.ReadString(path + ".md")
}

// --- Workspace Templates ---

// WorkspaceData is the data passed to workspace templates.
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
func ExtractWorkspace(tmplName, targetDir string, data *WorkspaceData) error {
	r := workspaceFS.Sub(tmplName)
	if !r.OK {
		if err, ok := r.Value.(error); ok {
			return err
		}
		return core.E("ExtractWorkspace", "template not found: "+tmplName, nil)
	}
	result := core.Extract(r.Value.(*core.Embed).FS(), targetDir, data)
	if !result.OK {
		if err, ok := result.Value.(error); ok {
			return err
		}
	}
	return nil
}

// --- List Functions ---

func ListPrompts() []string    { return listDir(promptFS) }
func ListFlows() []string      { return listDir(flowFS) }
func ListWorkspaces() []string { return listDir(workspaceFS) }

func ListTasks() []string {
	var slugs []string
	base := taskFS.BaseDirectory()
	fs.WalkDir(taskFS.FS(), base, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel := core.TrimPrefix(path, base+"/")
		ext := filepath.Ext(rel)
		slugs = append(slugs, core.TrimSuffix(rel, ext))
		return nil
	})
	return slugs
}

func ListPersonas() []string {
	var paths []string
	base := personaFS.BaseDirectory()
	fs.WalkDir(personaFS.FS(), base, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if core.HasSuffix(path, ".md") {
			rel := core.TrimPrefix(path, base+"/")
			rel = core.TrimSuffix(rel, ".md")
			paths = append(paths, rel)
		}
		return nil
	})
	return paths
}

func listDir(emb *core.Embed) []string {
	r := emb.ReadDir(".")
	if !r.OK {
		return nil
	}
	var slugs []string
	for _, e := range r.Value.([]fs.DirEntry) {
		name := e.Name()
		if e.IsDir() {
			slugs = append(slugs, name)
			continue
		}
		slugs = append(slugs, core.TrimSuffix(name, filepath.Ext(name)))
	}
	return slugs
}
