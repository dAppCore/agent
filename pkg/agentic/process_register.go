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

//	c := core.New(core.WithService(agentic.ProcessRegister))
//	processService, _ := core.ServiceFor[*process.Service](c, "process")
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
	service, ok := instance.(*process.Service)
	if !ok {
		return core.Result{Value: core.E("agentic.ProcessRegister", "unexpected process service type", nil), OK: false}
	}
	if registerResult := c.RegisterService("process", service); !registerResult.OK {
		return registerResult
	}

	handlers := &processActionHandlers{service: service}
	c.Action("process.run", handlers.handleRun)
	c.Action("process.start", handlers.handleStart)
	c.Action("process.kill", handlers.handleKill)

	return core.Result{OK: true}
}

func (h *processActionHandlers) handleRun(ctx context.Context, options core.Options) core.Result {
	output, err := h.service.RunWithOptions(ctx, process.RunOptions{
		Command: options.String("command"),
		Args:    optionStrings(options, "args"),
		Dir:     options.String("dir"),
		Env:     optionStrings(options, "env"),
	})
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

func (h *processActionHandlers) handleStart(ctx context.Context, options core.Options) core.Result {
	proc, err := h.service.StartWithOptions(ctx, process.RunOptions{
		Command: options.String("command"),
		Args:    optionStrings(options, "args"),
		Dir:     options.String("dir"),
		Env:     optionStrings(options, "env"),
		Detach:  options.Bool("detach"),
	})
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: proc, OK: true}
}

func (h *processActionHandlers) handleKill(_ context.Context, options core.Options) core.Result {
	id := options.String("id")
	if id == "" {
		return core.Result{Value: core.E("agentic.ProcessRegister", "process id is required", nil), OK: false}
	}
	if err := h.service.Kill(id); err != nil {
		return core.Result{Value: err, OK: false}
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
