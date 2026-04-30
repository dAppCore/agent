// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"syscall"

	core "dappco.re/go"
)

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

	validation := s.validateExecutableFlowDefinition(document)
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
	execution := s.executeFlowDefinition(document)
	output.Success = execution.Success
	output.Executed = execution.Executed
	output.Passed = execution.Passed
	output.Failed = execution.Failed
	output.StepResults = execution.StepResults

	printFlowExecutionSummary(output)
	return core.Result{Value: output, OK: output.Success}
}

func (s *PrepSubsystem) validateExecutableFlowDefinition(document flowRunDocument) core.Result {
	for index, step := range document.Definition.Steps {
		if err := validateExecutableFlowStep(s, index+1, step); err != nil {
			return core.Result{Value: err, OK: false}
		}
	}
	return core.Result{OK: true}
}

var validateExecutableFlowStep = func(s *PrepSubsystem, index int, step flowDefinitionStep) error {
	stepName := flowStepDisplayName(index, step)

	if core.Trim(step.Cmd) == "" {
		switch {
		case core.Trim(step.Flow) != "":
			return flowStepError(stepName, "cannot execute nested flow references; use flow/preview or convert to cmd")
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

func (s *PrepSubsystem) executeFlowDefinition(document flowRunDocument) flowExecutionSummary {
	summary := flowExecutionSummary{Success: true}

	for index, step := range document.Definition.Steps {
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
		break
	}

	return summary
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
		_ = syscall.Dup2(restoreStdoutFD, int(stdoutFile.Fd()))
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
		_ = syscall.Close(fd)
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
