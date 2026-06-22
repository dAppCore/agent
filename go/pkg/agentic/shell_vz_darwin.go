// SPDX-License-Identifier: EUPL-1.2

//go:build darwin

package agentic

import (
	"os"
	"os/signal"
	"syscall"

	core "dappco.re/go"
	"dappco.re/go/container"

	"golang.org/x/term"
)

// vzInteractiveShell drops the host terminal into a shell inside a running VZ
// guest. It puts the local terminal into raw mode (restored on every exit path,
// including panic, via defer), relays SIGWINCH window changes, and runs
// VZProvider.Shell over the vsock control channel. Requires a real TTY on stdin.
func vzInteractiveShell(id, shell string) core.Result {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return core.Fail(core.E("agentic.vzInteractiveShell", "stdin is not a terminal; an interactive shell needs a TTY", nil))
	}
	state, err := term.MakeRaw(fd)
	if err != nil {
		return core.Fail(core.E("agentic.vzInteractiveShell", "set terminal raw mode", err))
	}
	defer func() { _ = term.Restore(fd, state) }()

	cols, rows, err := term.GetSize(fd)
	if err != nil {
		cols, rows = 80, 24
	}

	resize := make(chan container.WinSize, 1)
	winch := make(chan os.Signal, 1)
	signal.Notify(winch, syscall.SIGWINCH)
	stopped := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(stopped)
		for {
			select {
			case <-done:
				return
			case <-winch:
				width, height, sizeErr := term.GetSize(fd)
				if sizeErr != nil {
					continue
				}
				select {
				case resize <- container.WinSize{Cols: width, Rows: height}:
				case <-done:
					return
				default: // a resize is already queued; drop this one
				}
			}
		}
	}()

	result := container.NewVZProvider().Shell(id, os.Stdin, os.Stdout, resize, container.WinSize{Cols: cols, Rows: rows}, shell)

	// Stop SIGWINCH and the relay before closing resize, so there is never a
	// send on a closed channel.
	signal.Stop(winch)
	close(done)
	<-stopped
	close(resize)
	return result
}
