// SPDX-License-Identifier: EUPL-1.2

// Package prompts re-exports the lib package for backwards compatibility.
// New code should import pkg/lib directly.
package prompts

import "forge.lthn.ai/core/agent/pkg/lib"

func Prompt(slug string) (string, error)                          { return lib.Prompt(slug) }
func Task(slug string) (string, error)                            { return lib.Task(slug) }
func TaskBundle(slug string) (string, map[string]string, error)   { return lib.TaskBundle(slug) }
func Flow(slug string) (string, error)                            { return lib.Flow(slug) }
func Persona(path string) (string, error)                         { return lib.Persona(path) }
func Template(slug string) (string, error)                        { return lib.Prompt(slug) }
func ListPrompts() []string                                       { return lib.ListPrompts() }
func ListTasks() []string                                         { return lib.ListTasks() }
func ListFlows() []string                                         { return lib.ListFlows() }
func ListPersonas() []string                                      { return lib.ListPersonas() }
func ListTemplates() []string                                     { return append(lib.ListPrompts(), lib.ListTasks()...) }
