package loop

import (
	"context"
	"fmt"
	"strings"

	"forge.lthn.ai/core/go-inference"
	coreerr "forge.lthn.ai/core/go-log"
)

// Engine drives the agent loop: prompt the model, parse tool calls, execute
// tools, feed results back, and repeat until the model responds without tool
// blocks or the turn limit is reached.
type Engine struct {
	model    inference.TextModel
	tools    []Tool
	system   string
	maxTurns int
}

// Option configures an Engine.
type Option func(*Engine)

// WithModel sets the inference backend for the engine.
func WithModel(m inference.TextModel) Option {
	return func(e *Engine) { e.model = m }
}

// WithTools registers tools that the model may invoke.
func WithTools(tools ...Tool) Option {
	return func(e *Engine) { e.tools = append(e.tools, tools...) }
}

// WithSystem overrides the default system prompt. When empty, BuildSystemPrompt
// generates one from the registered tools.
func WithSystem(prompt string) Option {
	return func(e *Engine) { e.system = prompt }
}

// WithMaxTurns caps the number of LLM calls before the loop errors out.
func WithMaxTurns(n int) Option {
	return func(e *Engine) { e.maxTurns = n }
}

// New creates an Engine with the given options. The default turn limit is 10.
func New(opts ...Option) *Engine {
	e := &Engine{maxTurns: 10}
	for _, o := range opts {
		o(e)
	}
	return e
}

// Run executes the agent loop. It sends userMessage to the model, parses any
// tool calls from the response, executes them, appends the results, and loops
// until the model produces a response with no tool blocks or maxTurns is hit.
func (e *Engine) Run(ctx context.Context, userMessage string) (*Result, error) {
	if e.model == nil {
		return nil, coreerr.E("loop.Run", "no model configured", nil)
	}

	system := e.system
	if system == "" {
		system = BuildSystemPrompt(e.tools)
	}

	handlers := make(map[string]func(context.Context, map[string]any) (string, error), len(e.tools))
	for _, tool := range e.tools {
		handlers[tool.Name] = tool.Handler
	}

	var history []Message
	history = append(history, Message{Role: RoleUser, Content: userMessage})

	for turn := 0; turn < e.maxTurns; turn++ {
		if err := ctx.Err(); err != nil {
			return nil, coreerr.E("loop.Run", "context cancelled", err)
		}

		prompt := BuildFullPrompt(system, history, "")
		var response strings.Builder
		for tok := range e.model.Generate(ctx, prompt) {
			response.WriteString(tok.Text)
		}
		if err := e.model.Err(); err != nil {
			return nil, coreerr.E("loop.Run", "inference error", err)
		}

		fullResponse := response.String()
		calls, cleanText := ParseToolCalls(fullResponse)

		history = append(history, Message{
			Role:     RoleAssistant,
			Content:  fullResponse,
			ToolUses: calls,
		})

		// No tool calls means the model has produced a final answer.
		if len(calls) == 0 {
			return &Result{
				Response: cleanText,
				Messages: history,
				Turns:    turn + 1,
			}, nil
		}

		// Execute each tool call and append results to the history.
		for _, call := range calls {
			handler, ok := handlers[call.Name]
			var resultText string
			if !ok {
				resultText = fmt.Sprintf("error: unknown tool %q", call.Name)
			} else {
				out, err := handler(ctx, call.Args)
				if err != nil {
					resultText = fmt.Sprintf("error: %v", err)
				} else {
					resultText = out
				}
			}

			history = append(history, Message{
				Role:     RoleToolResult,
				Content:  resultText,
				ToolUses: []ToolUse{{Name: call.Name}},
			})
		}
	}

	return nil, coreerr.E("loop.Run", fmt.Sprintf("max turns (%d) exceeded", e.maxTurns), nil)
}
