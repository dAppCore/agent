// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"

	core "dappco.re/go"
	command "dappco.re/go/process/exec"
)

// defaultShell is the in-container shell ContainerShell execs when the caller
// names none. /bin/sh is the one shell every OCI image and LinuxKit guest is
// guaranteed to ship — bash is routinely absent from minimal images, and the
// host's $SHELL is meaningless inside the guest, so it is never inherited.
const defaultShell = "/bin/sh"

// ShellRequest is the resolved input for ContainerShell.
//
//	ContainerShell(ShellRequest{ID: "vz-core-go-io-task-5"})
type ShellRequest struct {
	// ID is the running container/VM name to attach to (required).
	ID string
	// Runtime forces a runtime (apple|vz|docker|podman); empty resolves the
	// host's detected runtime — the same one dispatch would pick.
	Runtime string
	// Shell is the in-container shell to launch; empty uses /bin/sh.
	Shell string
}

// interactiveShellArgs builds the TTY-allocating `exec -i -t <id> <shell>` argv
// shared by the apple/docker/podman runtime CLIs. -t allocates a pseudo-terminal
// and -i keeps stdin open, so the runtime puts the local terminal into raw mode
// and restores it on exit — core-agent does not manage raw mode for the OCI path.
//
//	interactiveShellArgs("vz-core-go-io-task-5", "/bin/sh")
//	// []string{"exec", "-i", "-t", "vz-core-go-io-task-5", "/bin/sh"}
func interactiveShellArgs(id, shell string) []string {
	return []string{"exec", "-i", "-t", id, shell}
}

// ContainerShell drops the current terminal into an interactive shell inside a
// running container/VM. OCI runtimes (apple/docker/podman) exec
// `<rt> exec -i -t <id> <shell>` with the host stdio inherited, so the runtime
// CLI owns TTY raw-mode and restore. A VZ dispatch is answered with a clean
// not-yet-implemented error until the vsock PTY lane (SP4) lands — never a
// panic, so a caller on a VZ host gets a clear message rather than a crash.
//
//	r := ContainerShell(ShellRequest{ID: "vz-core-go-io-task-5"})
//	if !r.OK { core.Println(r.Error()) }
func ContainerShell(req ShellRequest) core.Result {
	id := core.Trim(req.ID)
	if id == "" {
		return core.Fail(core.E("agentic.ContainerShell", "container id is required", nil))
	}
	shell := core.Trim(req.Shell)
	if shell == "" {
		shell = defaultShell
	}

	runtime := core.Trim(req.Runtime)
	if runtime == "" || runtime == RuntimeAuto {
		runtime = resolveContainerRuntime(RuntimeAuto)
	}

	switch runtime {
	case RuntimeApple, RuntimeDocker, RuntimePodman:
		if !runtimeAvailable(runtime) {
			return core.Fail(core.E("agentic.ContainerShell", "container runtime not available: "+runtime, nil))
		}
		// The runtime CLI inherits the terminal and writes its own diagnostics,
		// so a non-zero shell exit — a normal end to an interactive session, or a
		// CLI error already shown on screen — is not re-surfaced as a verb
		// failure once the CLI has launched. (Propagating the shell's exit code
		// as core-agent's own process status is a tracked SP4 follow-up.)
		_ = runInteractiveExec(containerRuntimeBinary(runtime), interactiveShellArgs(id, shell))
		return core.Ok(nil)
	case RuntimeVZ:
		// The VZ guest has no container CLI — reach it over the vsock control
		// channel with a host-side raw terminal (darwin only).
		return vzInteractiveShell(id, shell)
	default:
		return core.Fail(core.E("agentic.ContainerShell", "unsupported runtime: "+runtime, nil))
	}
}

// runInteractiveExec runs binary+args attached to the host terminal — stdin,
// stdout and stderr are the process's real *os.File streams, so the runtime's
// -t flag detects a TTY and allocates one, and the child inherits the terminal
// until it exits. Run blocks for that lifetime; a non-zero child exit surfaces
// as a Fail carrying the exit code.
func runInteractiveExec(binary string, args []string) core.Result {
	return command.Command(context.Background(), binary, args...).
		WithStdin(core.Stdin()).
		WithStdout(core.Stdout()).
		WithStderr(core.Stderr()).
		Run()
}
