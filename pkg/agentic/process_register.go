// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"

	core "dappco.re/go"
	"dappco.re/go/process"
)

// processActionHandlers owns the agent-side overrides for the
// `process.*` actions. Start/run/kill return rich values (the
// `*process.Process` handle) instead of the raw string IDs surfaced
// by go-process so dispatch code can reap, signal, and tree-kill
// managed children without another lookup.
//
// Usage: `handlers := &processActionHandlers{service: svc}`
type processActionHandlers struct {
	service *process.Service
}

// ProcessRegister ensures a `*process.Service` is available under the
// "process" service name and installs the agent-specific action
// overrides.  Registering as a Startable service means the agent
// handlers run AFTER go-process's own OnStartup (which installs the
// string-ID variants), so the dispatch-friendly overrides always win.
//
// Usage:
//
//	c := core.New(core.WithService(agentic.ProcessRegister))
//	processService := c.Service("process")
func ProcessRegister(c *core.Core) core.Result {
	if c == nil {
		return core.Result{Value: core.E("agentic.ProcessRegister", "core is required", nil), OK: false}
	}

	var service *process.Service
	if result := c.Service("process"); result.OK {
		existing, ok := result.Value.(*process.Service)
		if !ok || existing == nil {
			return core.Result{Value: core.E("agentic.ProcessRegister", "unexpected process service type", nil), OK: false}
		}
		service = existing
	} else {
		factory := process.NewService(process.Options{})
		instanceResult := factory(c)
		if !instanceResult.OK {
			if err, ok := instanceResult.Value.(error); ok {
				return core.Result{Value: core.E("agentic.ProcessRegister", "create process service", err), OK: false}
			}
			return core.Result{Value: core.E("agentic.ProcessRegister", "create process service", nil), OK: false}
		}
		created, ok := instanceResult.Value.(*process.Service)
		if !ok {
			return core.Result{Value: core.E("agentic.ProcessRegister", "unexpected process service type", nil), OK: false}
		}
		service = created
		if registerResult := c.RegisterService("process", service); !registerResult.OK {
			return registerResult
		}
	}

	handlers := &processActionHandlers{service: service}
	// Install the overrides now — good for callers who never run
	// ServiceStartup (smaller test setups) and for the
	// pre-registered-service path where go-process may already have
	// started.
	handlers.registerActions(c)

	// Also register as a Startable service so the overrides survive
	// any subsequent `process` OnStartup that would otherwise
	// clobber them. The override service runs last because it
	// registers after `process`.
	overrideName := "agentic.process-overrides"
	if existing := c.Service(overrideName); !existing.OK {
		if registerResult := c.RegisterService(overrideName, &processOverrideService{handlers: handlers, core: c}); !registerResult.OK {
			return registerResult
		}
	}

	return core.Result{OK: true}
}

// processOverrideService reinstalls the agent-side action overrides
// once Core finishes calling OnStartup on every registered service.
// go-process re-registers `process.start`/`process.kill`/`process.run`
// during its own OnStartup, so the override has to run after that to
// keep the dispatch-friendly contract.
//
// Usage: `c.RegisterService("agentic.process-overrides", &processOverrideService{handlers: h, core: c})`
type processOverrideService struct {
	handlers *processActionHandlers
	core     *core.Core
}

// OnStartup is called by Core after every underlying service has
// booted. The override is reapplied at the tail of the lifecycle so
// the agent-side handlers win.
//
// Usage: `_ = svc.OnStartup(ctx)`
func (s *processOverrideService) OnStartup(context.Context) core.Result {
	if s == nil || s.handlers == nil {
		return core.Result{OK: true}
	}
	s.handlers.registerActions(s.core)
	return core.Result{OK: true}
}

// registerActions wires the override handlers onto `c`. It is safe
// to call multiple times — each call simply overwrites the same
// action names.
//
// Usage: `handlers.registerActions(c)`
func (h *processActionHandlers) registerActions(c *core.Core) {
	if h == nil || c == nil {
		return
	}
	c.Action("process.run", h.handleRun)
	c.Action("process.start", h.handleStart)
	c.Action("process.kill", h.handleKill)
}

func (h *processActionHandlers) handleRun(ctx context.Context, options core.Options) core.Result {
	outputResult := h.service.RunWithOptions(ctx, process.RunOptions{
		Command: options.String("command"),
		Args:    optionStrings(options, "args"),
		Dir:     options.String("dir"),
		Env:     optionStrings(options, "env"),
	})
	if !outputResult.OK {
		return outputResult
	}
	return outputResult
}

func (h *processActionHandlers) handleStart(ctx context.Context, options core.Options) core.Result {
	startResult := h.service.StartWithOptions(ctx, process.RunOptions{
		Command: options.String("command"),
		Args:    optionStrings(options, "args"),
		Dir:     options.String("dir"),
		Env:     optionStrings(options, "env"),
		Detach:  options.Bool("detach"),
	})
	if !startResult.OK {
		return startResult
	}
	return startResult
}

func (h *processActionHandlers) handleKill(_ context.Context, options core.Options) core.Result {
	id := options.String("id")
	if id == "" {
		return core.Result{Value: core.E("agentic.ProcessRegister", "process id is required", nil), OK: false}
	}
	if result := h.service.Kill(id); !result.OK {
		return result
	}
	return core.Result{OK: true}
}

func optionStrings(options core.Options, key string) []string {
	result := options.Get(key)
	if !result.OK {
		return nil
	}
	switch values := result.Value.(type) {
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
