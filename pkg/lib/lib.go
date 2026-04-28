// SPDX-License-Identifier: EUPL-1.2

// lib.MountData(c)
// prompts := lib.ListPrompts()
// task := lib.Task("code/review")
// persona := lib.Persona("secops/dev")
// data := &lib.WorkspaceData{Repo: "go-io", Task: "fix tests", Agent: "codex"}
// workspace := lib.ExtractWorkspace("default", "/tmp/workspace", data)
package lib

import (
	"embed"
	"sync/atomic"

	core "dappco.re/go"
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

	mountDone   atomic.Bool
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
	if mountDone.Load() {
		return mountResult
	}

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
			return mountResult
		}

		emb := mounted.Value.(*core.Embed)
		item.assign(emb)
		mountedData.Set(item.name, emb)
	}

	data = mountedData
	mountResult = core.Result{Value: mountedData, OK: true}
	mountDone.Store(true)

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

// r := lib.Template("coding")
// if r.OK { content := r.Value.(string) }
func Template(slug string) core.Result {
	if result := ensureMounted(); !result.OK {
		return result
	}
	if r := Prompt(slug); r.OK {
		return r
	}
	return Task(slug)
}

// r := lib.Prompt("coding")
// if r.OK { content := r.Value.(string) }
func Prompt(slug string) core.Result {
	if result := ensureMounted(); !result.OK {
		return result
	}
	return promptFS.ReadString(core.Concat(slug, ".md"))
}

// r := lib.Task("code/review")
// if r.OK { content := r.Value.(string) }
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

// bundle := lib.Bundle{Main: "Prompt body", Files: map[string]string{"README.md": "Context"}}
// core.Println(bundle.Files["README.md"])
type Bundle struct {
	Main  string
	Files map[string]string
}

// r := lib.TaskBundle("code/review")
// if r.OK { b := r.Value.(lib.Bundle) }
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

// r := lib.Flow("go")
// if r.OK { content := r.Value.(string) }
func Flow(slug string) core.Result {
	if result := ensureMounted(); !result.OK {
		return result
	}
	return flowFS.ReadString(core.Concat(slug, ".md"))
}

// r := lib.Persona("secops/developer")
// if r.OK { content := r.Value.(string) }
func Persona(path string) core.Result {
	if result := ensureMounted(); !result.OK {
		return result
	}
	return personaFS.ReadString(core.Concat(path, ".md"))
}

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

//	r := lib.ExtractWorkspace("default", "/tmp/ws", &lib.WorkspaceData{
//	    Repo: "go-io", Task: "fix tests", Agent: "codex",
//	})
//
// core.Println(r.OK)
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

// r := lib.WorkspaceFile("default", "CODEX-PHP.md.tmpl")
// if r.OK { content := r.Value.(string) }
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

// prompts := lib.ListPrompts() // ["coding", "conventions", "security"]
func ListPrompts() []string { return listNames("prompt") }

// flows := lib.ListFlows() // ["go", "php", "release"]
func ListFlows() []string { return listNames("flow") }

// templates := lib.ListWorkspaces() // ["default", "review", "security"]
func ListWorkspaces() []string { return listNames("workspace") }

// tasks := lib.ListTasks() // ["bug-fix", "code/review", "code/refactor"]
func ListTasks() []string {
	if result := ensureMounted(); !result.OK {
		return nil
	}

	result := listNamesRecursive("task", ".")
	names := core.NewArray(result...)
	names.Deduplicate()
	return names.AsSlice()
}

// personas := lib.ListPersonas() // ["code/backend-architect", "secops/security-developer", "testing/model-qa"]
func ListPersonas() []string {
	if result := ensureMounted(); !result.OK {
		return nil
	}

	names := core.NewArray(listNamesRecursive("persona", ".")...)
	names.Deduplicate()
	return names.AsSlice()
}

// names := listNamesRecursive("task", ".")
// core.Println(names) // ["bug-fix", "code/review", "code/refactor"]
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

		if childNames := data.ListNames(subPath); childNames.OK {
			slugs = append(slugs, listNamesRecursive(mount, relPath)...)
		}

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
