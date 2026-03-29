// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"

	core "dappco.re/go/core"
	"dappco.re/go/core/process"
)

type processActionHandlers struct {
	service *process.Service
}

// ProcessRegister is the service factory for go-process.
// Registers the process service under the canonical "process" name and exposes
// the Core `process.*` Actions expected by `c.Process()`.
//
//	core.New(
//	    core.WithService(agentic.ProcessRegister),
//	)
func ProcessRegister(c *core.Core) core.Result {
	if c == nil {
		return core.Result{Value: core.E("agentic.ProcessRegister", "core is required", nil), OK: false}
	}
	if _, ok := core.ServiceFor[*process.Service](c, "process"); ok {
		return core.Result{OK: true}
	}

	factory := process.NewService(process.Options{})
	instance, err := factory(c)
	if err != nil {
		return core.Result{Value: core.E("agentic.ProcessRegister", "create process service", err), OK: false}
	}
	svc, ok := instance.(*process.Service)
	if !ok {
		return core.Result{Value: core.E("agentic.ProcessRegister", "unexpected process service type", nil), OK: false}
	}
	if r := c.RegisterService("process", svc); !r.OK {
		return r
	}

	handlers := &processActionHandlers{service: svc}
	c.Action("process.run", handlers.handleRun)
	c.Action("process.start", handlers.handleStart)
	c.Action("process.kill", handlers.handleKill)

	return core.Result{OK: true}
}

func (h *processActionHandlers) handleRun(ctx context.Context, opts core.Options) core.Result {
	output, err := h.service.RunWithOptions(ctx, process.RunOptions{
		Command: opts.String("command"),
		Args:    optionStrings(opts, "args"),
		Dir:     opts.String("dir"),
		Env:     optionStrings(opts, "env"),
	})
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

func (h *processActionHandlers) handleStart(ctx context.Context, opts core.Options) core.Result {
	proc, err := h.service.StartWithOptions(ctx, process.RunOptions{
		Command: opts.String("command"),
		Args:    optionStrings(opts, "args"),
		Dir:     opts.String("dir"),
		Env:     optionStrings(opts, "env"),
		Detach:  opts.Bool("detach"),
	})
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: proc, OK: true}
}

func (h *processActionHandlers) handleKill(_ context.Context, opts core.Options) core.Result {
	id := opts.String("id")
	if id == "" {
		return core.Result{Value: core.E("agentic.ProcessRegister", "process id is required", nil), OK: false}
	}
	if err := h.service.Kill(id); err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{OK: true}
}

func optionStrings(opts core.Options, key string) []string {
	r := opts.Get(key)
	if !r.OK {
		return nil
	}
	switch values := r.Value.(type) {
	case []string:
		return values
	case []any:
		out := make([]string, 0, len(values))
		for _, value := range values {
			out = append(out, core.Sprint(value))
		}
		return out
	default:
		if text := core.Sprint(values); text != "" {
			return []string{text}
		}
	}
	return nil
}
