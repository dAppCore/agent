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

	// data wraps all embeds for ListNames access (avoids io/fs DirEntry import)
	data = newData()
)

func newData() *core.Data {
	d := &core.Data{Registry: core.NewRegistry[*core.Embed]()}
	d.Set("prompt", promptFS)
	d.Set("task", taskFS)
	d.Set("flow", flowFS)
	d.Set("persona", personaFS)
	d.Set("workspace", workspaceFS)
	return d
}

// MountData registers all embedded content (prompts, tasks, flows, personas, workspaces)
// into Core's Data registry. Other services can then access content without importing lib:
//
//	lib.MountData(c)
//	r := c.Data().ReadString("prompts/coding.md")
//	r := c.Data().ListNames("flows")
func MountData(c *core.Core) {
	d := c.Data()
	d.New(core.NewOptions(
		core.Option{Key: "name", Value: "prompts"},
		core.Option{Key: "source", Value: promptFiles},
		core.Option{Key: "path", Value: "prompt"},
	))
	d.New(core.NewOptions(
		core.Option{Key: "name", Value: "tasks"},
		core.Option{Key: "source", Value: taskFiles},
		core.Option{Key: "path", Value: "task"},
	))
	d.New(core.NewOptions(
		core.Option{Key: "name", Value: "flows"},
		core.Option{Key: "source", Value: flowFiles},
		core.Option{Key: "path", Value: "flow"},
	))
	d.New(core.NewOptions(
		core.Option{Key: "name", Value: "personas"},
		core.Option{Key: "source", Value: personaFiles},
		core.Option{Key: "path", Value: "persona"},
	))
	d.New(core.NewOptions(
		core.Option{Key: "name", Value: "workspaces"},
		core.Option{Key: "source", Value: workspaceFiles},
		core.Option{Key: "path", Value: "workspace"},
	))
}

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
	return promptFS.ReadString(core.Concat(slug, ".md"))
}

// Task reads a structured task plan by slug. Tries .md, .yaml, .yml.
//
//	r := lib.Task("code/review")
//	if r.OK { content := r.Value.(string) }
func Task(slug string) core.Result {
	for _, ext := range []string{".md", ".yaml", ".yml"} {
		if r := taskFS.ReadString(core.Concat(slug, ext)); r.OK {
			return r
		}
	}
	return core.Result{OK: false}
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
	nr := data.ListNames(core.Concat("task/", slug))
	if nr.OK {
		for _, name := range nr.Value.([]string) {
			for _, ext := range []string{".md", ".yaml", ".yml", ".txt", ""} {
				fullName := core.Concat(name, ext)
				if fr := taskFS.ReadString(core.Concat(slug, "/", fullName)); fr.OK {
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
	return flowFS.ReadString(core.Concat(slug, ".md"))
}

// Persona reads a domain/role persona by path.
//
//	r := lib.Persona("secops/developer")
//	if r.OK { content := r.Value.(string) }
func Persona(path string) core.Result {
	return personaFS.ReadString(core.Concat(path, ".md"))
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
//
//	lib.ExtractWorkspace("default", "/tmp/ws", &lib.WorkspaceData{
//	    Repo: "go-io", Task: "fix tests", Agent: "codex",
//	})
func ExtractWorkspace(tmplName, targetDir string, data *WorkspaceData) error {
	r := workspaceFS.Sub(tmplName)
	if !r.OK {
		if err, ok := r.Value.(error); ok {
			return err
		}
		return core.E("ExtractWorkspace", core.Concat("template not found: ", tmplName), nil)
	}
	result := core.Extract(r.Value.(*core.Embed).FS(), targetDir, data)
	if !result.OK {
		if err, ok := result.Value.(error); ok {
			return err
		}
	}
	return nil
}

// WorkspaceFile reads a single file from a workspace template.
// Returns the file content as a string.
//
//	r := lib.WorkspaceFile("default", "CODEX-PHP.md.tmpl")
//	if r.OK { content := r.Value.(string) }
func WorkspaceFile(tmplName, filename string) core.Result {
	r := workspaceFS.Sub(tmplName)
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
	result := listNamesRecursive("task", taskFS, ".")
	a := core.NewArray(result...)
	a.Deduplicate()
	return a.AsSlice()
}

// ListPersonas returns available persona paths, including nested directories.
//
//	personas := lib.ListPersonas() // ["code/go", "secops/developer", ...]
func ListPersonas() []string {
	a := core.NewArray(listNamesRecursive("persona", personaFS, ".")...)
	a.Deduplicate()
	return a.AsSlice()
}


// listNamesRecursive walks an embed tree via Data.ListNames.
// Directories are recursed into. Files are added as slugs (extension stripped by ListNames).
// A name can be both a file AND a directory (e.g. code/review.md + code/review/).
func listNamesRecursive(mount string, emb *core.Embed, dir string) []string {
	path := core.Concat(mount, "/", dir)
	nr := data.ListNames(path)
	if !nr.OK {
		return nil
	}

	var slugs []string
	for _, name := range nr.Value.([]string) {
		relPath := name
		if dir != "." {
			relPath = core.Concat(dir, "/", name)
		}

		subPath := core.Concat(mount, "/", relPath)

		// Try as directory — recurse if it has contents
		if sub := data.ListNames(subPath); sub.OK {
			slugs = append(slugs, listNamesRecursive(mount, emb, relPath)...)
		}

		// Always add the slug — ListNames includes both files and dirs
		slugs = append(slugs, relPath)
	}
	return slugs
}

func listNames(mount string) []string {
	r := data.ListNames(core.Concat(mount, "/."))
	if !r.OK {
		return nil
	}
	return r.Value.([]string)
}
