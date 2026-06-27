<!-- SPDX-License-Identifier: EUPL-1.2 -->
# Container shell

Drop an interactive terminal into a running dispatch container or VM — useful for
inspecting what a containerised runner ([codex/gemini](../dispatch/)) is doing.

```bash
core-agent shell <id> [--runtime=<rt>] [--shell=<path>]
```

- `<id>` — the container/VM to attach to.
- `--runtime` — `apple` (VZ), `docker`, or `podman`; defaults to the resolved runtime
  (unknown ⇒ `docker`).
- `--shell` — the shell binary to exec (default the container's login shell).

It **attaches your current terminal** to the running container (`ExampleContainerShell`);
on the Apple/VZ path it goes through `vzInteractiveShell(id, shell)`. This is the
container side of VZ-first dispatch — the same runtimes [dispatch](../dispatch/) uses to
run codex/gemini.

## Next

[dispatch](../dispatch/) (where the containers come from) · [cli](../cli/) (`shell`).
