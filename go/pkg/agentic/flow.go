// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"syscall"

	core "dappco.re/go"
	"dappco.re/go/agent/pkg/lib/flow"
)

// maxFlowNestingDepth bounds how deeply a flow may compose other flows at
// run time. A flow that references another flow that references another flow
// (and so on) is expanded inline; this guard rejects pathological nesting
// before it can exhaust the stack. The root flow is depth 0, its direct
// nested children depth 1, and so on.
const maxFlowNestingDepth = 16

// FlowRunStepOutput captures the per-step result of a flow execution: the
// step name, command + args, exit success, and stdout/stderr/error tail.
// Returned in slices from Flow runners so callers can inspect each step.
//
//	out := FlowRunStepOutput{Name: "build", Command: "task", Success: true}
type FlowRunStepOutput struct {
	Name            string   `json:"name,omitempty"`
	Command         string   `json:"command,omitempty"`
	Args            []string `json:"args,omitempty"`
	Success         bool     `json:"success"`
	ContinueOnError bool     `json:"continue_on_error,omitempty"`
	Stdout          string   `json:"stdout,omitempty"`
	Stderr          string   `json:"stderr,omitempty"`
	Error           string   `json:"error,omitempty"`
}

type flowExecutionSummary struct {
	Success     bool
	Executed    int
	Passed      int
	Failed      int
	StepResults []FlowRunStepOutput
}

const flowRunCommandContext = "agentic.cmdRunFlow"

func (s *PrepSubsystem) runFlowExecutionCommand(options core.Options, commandLabel string) core.Result {
	if optionBoolValue(options, "dry_run", "dry-run") {
		return s.runFlowCommand(options, commandLabel)
	}

	flowPath := optionStringValue(options, "_arg", `path`, "slug")
	if flowPath == "" {
		core.Print(nil, "usage: core-agent %s <path-or-slug> [--dry-run] [--var=key=value] [--vars='{\"key\":\"value\"}'] [--variables='{\"key\":\"value\"}']", commandLabel)
		return core.Result{Value: core.E(flowRunCommandContext, "flow path or slug is required", nil), OK: false}
	}

	variables := optionStringMapValue(options, "var", "vars", "variables")
	flowResult := readFlowDocument(flowPath, variables)
	if !flowResult.OK {
		core.Print(nil, "error: %v", flowResult.Value)
		return core.Result{Value: flowResult.Value, OK: false}
	}

	document, ok := flowResult.Value.(flowRunDocument)
	if !ok || !document.Parsed {
		err := core.E(flowRunCommandContext, "invalid flow definition", nil)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	validation := s.validateExecutableFlowDefinition(document, variables)
	if !validation.OK {
		err, ok := validation.Value.(error)
		if !ok {
			err = core.E(flowRunCommandContext, "invalid flow definition", nil)
		}
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	output := FlowRunOutput{
		Success:     true,
		Source:      document.Source,
		Name:        document.Definition.Name,
		Description: document.Definition.Description,
		Steps:       len(document.Definition.Steps),
		Parsed:      document.Parsed,
	}

	core.Print(nil, "flow:  %s", document.Source)
	if len(variables) > 0 {
		core.Print(nil, "vars: %d", len(variables))
	}
	if output.Name != "" {
		core.Print(nil, "name:  %s", output.Name)
	}
	if output.Description != "" {
		core.Print(nil, "desc:  %s", output.Description)
	}

	if len(document.Definition.Steps) == 0 {
		core.Print(nil, "steps: 0")
		printFlowExecutionSummary(output)
		return core.Result{Value: output, OK: true}
	}

	core.Print(nil, "steps: %d", len(document.Definition.Steps))
	rootCtx := flowExpansionContext{
		visited:   map[string]bool{document.Source: true},
		depth:     0,
		variables: variables,
	}
	execution := s.executeFlowDefinition(document, rootCtx)
	output.Success = execution.Success
	output.Executed = execution.Executed
	output.Passed = execution.Passed
	output.Failed = execution.Failed
	output.StepResults = execution.StepResults

	printFlowExecutionSummary(output)
	return core.Result{Value: output, OK: output.Success}
}

// flowExpansionContext threads the cycle-detection set, current nesting depth,
// and template variables through nested-flow validation and execution so a
// flow that composes another flow (Mantis #1805) is expanded inline with
// cycle + depth guards.
//
//	ctx := flowExpansionContext{visited: map[string]bool{src: true}, variables: vars}
type flowExpansionContext struct {
	visited   map[string]bool
	depth     int
	variables map[string]string
}

func (s *PrepSubsystem) validateExecutableFlowDefinition(document flowRunDocument, variables map[string]string) core.Result {
	ctx := flowExpansionContext{
		visited:   map[string]bool{document.Source: true},
		depth:     0,
		variables: variables,
	}
	if err := s.validateExecutableFlowSteps(document, ctx); err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{OK: true}
}

func (s *PrepSubsystem) validateExecutableFlowSteps(document flowRunDocument, ctx flowExpansionContext) error {
	for index, step := range document.Definition.Steps {
		if err := s.validateExecutableFlowStep(index+1, step, document.Source, ctx); err != nil {
			return err
		}
	}
	return nil
}

func (s *PrepSubsystem) validateExecutableFlowStep(index int, step flowDefinitionStep, baseSource string, ctx flowExpansionContext) error {
	stepName := flowStepDisplayName(index, step)

	if core.Trim(step.Cmd) == "" {
		switch {
		case core.Trim(step.Flow) != "":
			return s.validateNestedFlowStep(stepName, step, baseSource, ctx)
		case core.Trim(step.Run) != "":
			return flowStepError(stepName, "uses legacy run syntax; use cmd and args")
		default:
			return flowStepError(stepName, "must define cmd")
		}
	}

	commandResult := s.Core().Command(step.Cmd)
	if !commandResult.OK {
		return flowStepError(stepName, core.Concat("references unknown command: ", step.Cmd))
	}

	command, ok := commandResult.Value.(*core.Command)
	if !ok || command == nil || command.Action == nil {
		return flowStepError(stepName, core.Concat("references a non-executable command: ", step.Cmd))
	}

	return nil
}

// validateNestedFlowStep resolves the nested flow a step references, rejects
// composition that would exceed the depth guard or form a cycle, validates the
// supplied `with` args against the nested flow's declared Inputs (Mantis
// #1804), then recurses into the nested flow's own steps.
func (s *PrepSubsystem) validateNestedFlowStep(stepName string, step flowDefinitionStep, baseSource string, ctx flowExpansionContext) error {
	if ctx.depth+1 > maxFlowNestingDepth {
		return flowStepError(stepName, core.Concat("nested flow depth exceeds limit of ", core.Itoa(maxFlowNestingDepth)))
	}

	resolved := s.resolveFlowReference(baseSource, step.Flow, ctx.variables)
	if !resolved.OK {
		if err, ok := resolved.Value.(error); ok {
			return flowStepError(stepName, core.Concat("references unresolvable flow: ", err.Error()))
		}
		return flowStepError(stepName, core.Concat("references unresolvable flow: ", step.Flow))
	}

	nested, ok := resolved.Value.(flowRunDocument)
	if !ok || !nested.Parsed {
		return flowStepError(stepName, core.Concat("references a non-flow document: ", step.Flow))
	}

	if ctx.visited[nested.Source] {
		return flowStepError(stepName, core.Concat("forms a flow cycle: ", nested.Source))
	}

	if err := validateNestedFlowInputs(stepName, nested.Definition, step.With); err != nil {
		return err
	}

	childCtx := ctx.descend(nested.Source)
	return s.validateExecutableFlowSteps(nested, childCtx)
}

// validateNestedFlowInputs checks the args a parent step passes into a nested
// flow against that flow's declared Inputs schema, reusing the #1804
// flow.Flow.ValidateInputs implementation rather than duplicating it.
func validateNestedFlowInputs(stepName string, definition flowDefinition, with map[string]string) error {
	if len(definition.Inputs) == 0 {
		return nil
	}
	schema := flow.Flow{Inputs: definition.Inputs}
	if err := schema.ValidateInputs(with); err != nil {
		return flowStepError(stepName, core.Concat("nested flow input invalid: ", err.Error()))
	}
	return nil
}

// descend returns a child expansion context: the nested flow source added to
// the cycle-detection set and the depth incremented by one. The parent's set
// is copied so sibling branches do not see each other's in-progress sources.
func (ctx flowExpansionContext) descend(source string) flowExpansionContext {
	visited := make(map[string]bool, len(ctx.visited)+1)
	for key := range ctx.visited {
		visited[key] = true
	}
	visited[source] = true
	return flowExpansionContext{
		visited:   visited,
		depth:     ctx.depth + 1,
		variables: ctx.variables,
	}
}

func (s *PrepSubsystem) executeFlowDefinition(document flowRunDocument, ctx flowExpansionContext) flowExecutionSummary {
	summary := flowExecutionSummary{Success: true}
	s.accumulateFlowExecution(&summary, document, ctx)
	return summary
}

// accumulateFlowExecution runs each step of a flow into the shared summary,
// recursing into nested flow references (Mantis #1805) so their steps execute
// inline. Returns false when a non-continue failure should abort the parent.
func (s *PrepSubsystem) accumulateFlowExecution(summary *flowExecutionSummary, document flowRunDocument, ctx flowExpansionContext) bool {
	for index, step := range document.Definition.Steps {
		if core.Trim(step.Cmd) == "" && core.Trim(step.Flow) != "" {
			if !s.executeNestedFlowStep(summary, index+1, step, document.Source, ctx) {
				return false
			}
			continue
		}

		stepOutput := s.executeFlowStep(index+1, step)
		summary.Executed++
		summary.StepResults = append(summary.StepResults, stepOutput)

		if stepOutput.Success {
			summary.Passed++
			continue
		}

		summary.Failed++
		if stepOutput.ContinueOnError {
			continue
		}

		summary.Success = false
		return false
	}

	return true
}

// executeNestedFlowStep resolves a step's flow reference and executes the
// nested flow's steps inline. Validation (cycle, depth, inputs) has already
// run in validateExecutableFlowDefinition, so resolution failures here are
// treated as a failed step honouring continueOnError. Returns false when the
// parent flow should abort.
func (s *PrepSubsystem) executeNestedFlowStep(summary *flowExecutionSummary, index int, step flowDefinitionStep, baseSource string, ctx flowExpansionContext) bool {
	stepName := flowStepDisplayName(index, step)

	resolved := s.resolveFlowReference(baseSource, step.Flow, ctx.variables)
	nested, ok := resolved.Value.(flowRunDocument)
	if !resolved.OK || !ok || !nested.Parsed {
		summary.Executed++
		summary.Failed++
		summary.StepResults = append(summary.StepResults, FlowRunStepOutput{
			Name:            stepName,
			ContinueOnError: step.ContinueOnError,
			Error:           flowStepError(stepName, core.Concat("references unresolvable flow: ", step.Flow)).Error(),
		})
		if step.ContinueOnError {
			return true
		}
		summary.Success = false
		return false
	}

	core.Print(nil, "%d. %s", index, flowStepSummary(step))
	core.Print(nil, "  resolved: %s", nested.Source)

	childCtx := ctx.descend(nested.Source)
	return s.accumulateFlowExecution(summary, nested, childCtx)
}

func (s *PrepSubsystem) executeFlowStep(index int, step flowDefinitionStep) FlowRunStepOutput {
	stepOutput := FlowRunStepOutput{
		Name:            flowStepDisplayName(index, step),
		Command:         step.Cmd,
		Args:            append([]string(nil), step.Args...),
		ContinueOnError: step.ContinueOnError,
	}

	core.Print(nil, "%d. %s", index, flowStepSummary(step))

	command := s.Core().Command(step.Cmd).Value.(*core.Command)
	result, stdout, stderr, err := captureFlowStepOutput(func() core.Result {
		return command.Run(flowStepOptions(step.Args))
	})

	if stdout != "" {
		stepOutput.Stdout = stdout
		printFlowStepStream("stdout", stdout)
	}
	if stderr != "" {
		stepOutput.Stderr = stderr
		printFlowStepStream("stderr", stderr)
	}

	if err != nil {
		stepOutput.Error = err.Error()
		core.Print(nil, "  status: failed")
		core.Print(nil, "  error:  %s", stepOutput.Error)
		return stepOutput
	}

	stepOutput.Success = result.OK
	if result.OK {
		core.Print(nil, "  status: passed")
		return stepOutput
	}

	stepOutput.Error = commandResultError(flowRunCommandContext, result).Error()
	if stepOutput.ContinueOnError {
		core.Print(nil, "  status: failed (continued)")
	} else {
		core.Print(nil, "  status: failed")
	}
	core.Print(nil, "  error:  %s", stepOutput.Error)
	return stepOutput
}

func flowStepDisplayName(index int, step flowDefinitionStep) string {
	if name := core.Trim(step.Name); name != "" {
		return name
	}
	if name := core.Trim(step.Cmd); name != "" {
		return name
	}
	if name := core.Trim(step.Flow); name != "" {
		return name
	}
	if name := core.Trim(step.Run); name != "" {
		return name
	}
	return core.Concat("step-", core.Itoa(index))
}

var flowStepError = func(stepName, message string) error {
	return core.E(
		"agentic.validateExecutableFlowStep",
		core.Concat("step \"", stepName, "\" ", message),
		nil,
	)
}

func flowStepCommandLine(step flowDefinitionStep) string {
	command := core.Trim(step.Cmd)
	if len(step.Args) == 0 {
		return command
	}
	return core.Concat(command, " ", core.Join(" ", step.Args...))
}

func flowStepOptions(args []string) core.Options {
	options := core.NewOptions()
	for _, arg := range args {
		key, value, ok := core.ParseFlag(arg)
		if ok {
			if core.Contains(arg, "=") {
				options.Set(key, value)
			} else {
				options.Set(key, true)
			}
			continue
		}

		if !core.IsFlag(arg) {
			options.Set("_arg", arg)
		}
	}
	return options
}

var captureFlowStepOutput = func(run func() core.Result) (core.Result, string, string, error) {
	stdoutReadFD, stdoutWriteFD, err := newPipe()
	if err != nil {
		return core.Result{}, "", "", core.E("agentic.captureFlowStepOutput", "create stdout pipe", err)
	}
	stderrReadFD, stderrWriteFD, err := newPipe()
	if err != nil {
		closeFD(stdoutReadFD)
		closeFD(stdoutWriteFD)
		return core.Result{}, "", "", core.E("agentic.captureFlowStepOutput", "create stderr pipe", err)
	}

	stdoutFile, ok := core.Stdout().(*core.OSFile)
	if !ok {
		closeFD(stdoutReadFD)
		closeFD(stdoutWriteFD)
		closeFD(stderrReadFD)
		closeFD(stderrWriteFD)
		return core.Result{}, "", "", core.E("agentic.captureFlowStepOutput", "stdout is not a file", nil)
	}
	stderrFile, ok := core.Stderr().(*core.OSFile)
	if !ok {
		closeFD(stdoutReadFD)
		closeFD(stdoutWriteFD)
		closeFD(stderrReadFD)
		closeFD(stderrWriteFD)
		return core.Result{}, "", "", core.E("agentic.captureFlowStepOutput", "stderr is not a file", nil)
	}

	restoreStdoutFD, err := syscall.Dup(int(stdoutFile.Fd()))
	if err != nil {
		closeFD(stdoutReadFD)
		closeFD(stdoutWriteFD)
		closeFD(stderrReadFD)
		closeFD(stderrWriteFD)
		return core.Result{}, "", "", core.E("agentic.captureFlowStepOutput", "dup stdout", err)
	}
	restoreStderrFD, err := syscall.Dup(int(stderrFile.Fd()))
	if err != nil {
		closeFD(stdoutReadFD)
		closeFD(stdoutWriteFD)
		closeFD(stderrReadFD)
		closeFD(stderrWriteFD)
		closeFD(restoreStdoutFD)
		return core.Result{}, "", "", core.E("agentic.captureFlowStepOutput", "dup stderr", err)
	}
	if err := syscall.Dup2(stdoutWriteFD, int(stdoutFile.Fd())); err != nil {
		closeFD(stdoutReadFD)
		closeFD(stdoutWriteFD)
		closeFD(stderrReadFD)
		closeFD(stderrWriteFD)
		closeFD(restoreStdoutFD)
		closeFD(restoreStderrFD)
		return core.Result{}, "", "", core.E("agentic.captureFlowStepOutput", "redirect stdout", err)
	}
	if err := syscall.Dup2(stderrWriteFD, int(stderrFile.Fd())); err != nil {
		if restoreErr := syscall.Dup2(restoreStdoutFD, int(stdoutFile.Fd())); restoreErr != nil {
			return core.Result{}, "", "", core.E("agentic.captureFlowStepOutput", "restore stdout", restoreErr)
		}
		closeFD(stdoutReadFD)
		closeFD(stdoutWriteFD)
		closeFD(stderrReadFD)
		closeFD(stderrWriteFD)
		closeFD(restoreStdoutFD)
		closeFD(restoreStderrFD)
		return core.Result{}, "", "", core.E("agentic.captureFlowStepOutput", "redirect stderr", err)
	}

	result := run()

	restoreErr := syscall.Dup2(restoreStdoutFD, int(stdoutFile.Fd()))
	if stderrRestoreErr := syscall.Dup2(restoreStderrFD, int(stderrFile.Fd())); restoreErr == nil {
		restoreErr = stderrRestoreErr
	}
	closeFD(restoreStdoutFD)
	closeFD(restoreStderrFD)
	closeFD(stdoutWriteFD)
	closeFD(stderrWriteFD)
	if restoreErr != nil {
		closeFD(stdoutReadFD)
		closeFD(stderrReadFD)
		return result, "", "", core.E("agentic.captureFlowStepOutput", "restore stdio", restoreErr)
	}

	stdoutData, err := readFD(stdoutReadFD)
	if err != nil {
		closeFD(stdoutReadFD)
		closeFD(stderrReadFD)
		return result, "", "", core.E("agentic.captureFlowStepOutput", "read stdout pipe", err)
	}
	stderrData, err := readFD(stderrReadFD)
	if err != nil {
		closeFD(stdoutReadFD)
		closeFD(stderrReadFD)
		return result, "", "", core.E("agentic.captureFlowStepOutput", "read stderr pipe", err)
	}

	closeFD(stdoutReadFD)
	closeFD(stderrReadFD)
	return result, string(stdoutData), string(stderrData), nil
}

var newPipe = func() (int, int, error) {
	var fds [2]int
	if err := syscall.Pipe(fds[:]); err != nil {
		return 0, 0, err
	}
	return fds[0], fds[1], nil
}

var readFD = func(fd int) ([]byte, error) {
	buffer := core.NewBuffer()
	chunk := make([]byte, 4096)
	for {
		n, err := syscall.Read(fd, chunk)
		if n > 0 {
			if _, writeErr := buffer.Write(chunk[:n]); writeErr != nil {
				return nil, writeErr
			}
		}
		if err != nil {
			if err == syscall.EINTR {
				continue
			}
			return nil, err
		}
		if n == 0 {
			return append([]byte(nil), buffer.Bytes()...), nil
		}
	}
}

func closeFD(fd int) {
	if fd > 0 {
		if err := syscall.Close(fd); err != nil {
			core.Warn("agentic.flow: close fd failed", "fd", fd, "reason", err)
		}
	}
}

func printFlowStepStream(label, stream string) {
	trimmed := core.TrimSuffix(core.Replace(stream, "\r\n", "\n"), "\n")
	if trimmed == "" {
		return
	}

	core.Print(nil, "  %s:", label)
	for _, line := range core.Split(trimmed, "\n") {
		core.Print(nil, "    %s", line)
	}
}

func printFlowExecutionSummary(output FlowRunOutput) {
	core.Print(nil, "")
	core.Print(nil, "summary:")

	for _, step := range output.StepResults {
		status := "passed"
		if !step.Success {
			if step.ContinueOnError {
				status = "failed (continued)"
			} else {
				status = "failed"
			}
		}
		core.Print(nil, "  %s: %s", step.Name, status)
	}

	core.Print(nil, "totals: ran=%d passed=%d failed=%d", output.Executed, output.Passed, output.Failed)
}
