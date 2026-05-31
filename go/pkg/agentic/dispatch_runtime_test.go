// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	core "dappco.re/go"
)

// --- containerRuntimeBinary ---

func TestDispatchRuntime_ContainerRuntimeBinary_Good_Case(t *testing.T) {
	core.AssertEqual(t, "container", containerRuntimeBinary(RuntimeApple))
	core.AssertEqual(t, "docker", containerRuntimeBinary(RuntimeDocker))
	core.AssertEqual(t, "podman", containerRuntimeBinary(RuntimePodman))
}

func TestDispatchRuntime_ContainerRuntimeBinary_Bad_Case(t *testing.T) {
	// Unknown runtime falls back to docker so dispatch never silently breaks.
	empty := containerRuntimeBinary("")
	unknown := containerRuntimeBinary("kubernetes")
	core.AssertEqual(t, "docker", empty)
	core.AssertEqual(t, "docker", unknown)
}

func TestDispatchRuntime_ContainerRuntimeBinary_Ugly_Case(t *testing.T) {
	// Whitespace-laden runtime name is treated as unknown; docker fallback wins.
	runtimeName := "  apple  "
	result := containerRuntimeBinary(runtimeName)
	core.AssertEqual(t, "docker", result)
	core.AssertNotEqual(t, "container", result)
}

// --- runtimeAvailable ---

func TestDispatchRuntime_RuntimeAvailable_Good_Case(t *testing.T) {
	// Inspect only the failure path that doesn't depend on host binaries.
	// Apple Container is by definition unavailable on non-darwin.
	if !isDarwin() {
		core.AssertFalse(t, runtimeAvailable(RuntimeApple))
	}
}

func TestDispatchRuntime_RuntimeAvailable_Bad_Case(t *testing.T) {
	// Unknown runtimes are never available.
	empty := runtimeAvailable("")
	unknown := runtimeAvailable("kubernetes")
	core.AssertFalse(t, empty)
	core.AssertFalse(t, unknown)
}

func TestDispatchRuntime_RuntimeAvailable_Ugly_Case(t *testing.T) {
	// Apple Container on non-macOS hosts is always unavailable, regardless of
	// whether a binary called "container" happens to be on PATH.
	if !isDarwin() {
		core.AssertFalse(t, runtimeAvailable(RuntimeApple))
	}
}

// --- resolveContainerRuntime ---

func TestDispatchRuntime_ResolveContainerRuntime_Good_Case(t *testing.T) {
	// Empty preference falls back to one of the known runtimes (docker is the
	// hard fallback, but the function may surface apple/podman when those
	// binaries exist on the test host).
	resolved := resolveContainerRuntime("")
	core.AssertNotEmpty(t, resolved)
	core.AssertContains(t, []string{RuntimeApple, RuntimeDocker, RuntimePodman}, resolved)
}

func TestDispatchRuntime_ResolveContainerRuntime_Bad_Case(t *testing.T) {
	// An unknown runtime preference still resolves to a known runtime.
	resolved := resolveContainerRuntime("kubernetes")
	core.AssertNotEmpty(t, resolved)
	core.AssertContains(t, []string{RuntimeApple, RuntimeDocker, RuntimePodman}, resolved)
}

func TestDispatchRuntime_ResolveContainerRuntime_Ugly_Case(t *testing.T) {
	// Apple preference on non-darwin host falls back to a non-apple runtime.
	if !isDarwin() {
		resolved := resolveContainerRuntime(RuntimeApple)
		core.AssertNotEqual(t, RuntimeApple, resolved)
	}
}

// --- containerCommandFor ---

func TestDispatchRuntime_ContainerCommandFor_Good_Case(t *testing.T) {
	t.Setenv("AGENT_DOCKER_IMAGE", "")
	t.Setenv("DIR_HOME", "/home/dev")

	// Docker runtime emits docker binary and includes host-gateway alias.
	cmd, args := containerCommandFor(RuntimeDocker, "core-dev", false, "codex", []string{"exec"}, "/ws", "/ws/.meta")
	core.AssertEqual(t, "docker", cmd)
	joined := core.Join(" ", args...)
	core.AssertContains(t, joined, "--add-host=host.docker.internal:host-gateway")
	core.AssertContains(t, joined, "core-dev")
}

func TestDispatchRuntime_ContainerCommandFor_Bad_Case(t *testing.T) {
	t.Setenv("AGENT_DOCKER_IMAGE", "")
	t.Setenv("DIR_HOME", "/home/dev")

	// Empty image resolves to the default rather than passing "" to docker.
	cmd, args := containerCommandFor(RuntimeDocker, "", false, "codex", nil, "/ws", "/ws/.meta")
	core.AssertEqual(t, "docker", cmd)
	core.AssertContains(t, args, defaultDockerImage)
}

func TestDispatchRuntime_ContainerCommandFor_Ugly_Case(t *testing.T) {
	t.Setenv("AGENT_DOCKER_IMAGE", "")
	t.Setenv("DIR_HOME", "/home/dev")

	// Apple runtime emits the `container` binary and SKIPS the host-gateway
	// alias because Apple Containers don't support `--add-host=host-gateway`.
	cmd, args := containerCommandFor(RuntimeApple, "core-dev", false, "codex", []string{"exec"}, "/ws", "/ws/.meta")
	core.AssertEqual(t, "container", cmd)
	joined := core.Join(" ", args...)
	core.AssertNotContains(t, joined, "--add-host=host.docker.internal:host-gateway")

	// Podman runtime emits the `podman` binary.
	cmd2, _ := containerCommandFor(RuntimePodman, "core-dev", false, "codex", []string{"exec"}, "/ws", "/ws/.meta")
	core.AssertEqual(t, "podman", cmd2)

	// GPU passthrough on docker emits `--gpus=all`.
	_, gpuArgs := containerCommandFor(RuntimeDocker, "core-dev", true, "codex", []string{"exec"}, "/ws", "/ws/.meta")
	core.AssertContains(t, core.Join(" ", gpuArgs...), "--gpus=all")

	// GPU passthrough on apple emits `--gpu=metal` for Metal passthrough.
	_, appleGPUArgs := containerCommandFor(RuntimeApple, "core-dev", true, "codex", []string{"exec"}, "/ws", "/ws/.meta")
	core.AssertContains(t, core.Join(" ", appleGPUArgs...), "--gpu=metal")
}

// --- containerCommandFor: opencode credential scratch mount ---

func opencodeTestSeedCredential(t *testing.T, home string) {
	t.Helper()
	dataDir := core.JoinPath(home, ".local", "share", "opencode")
	core.RequireTrue(t, fs.EnsureDir(dataDir).OK)
	core.RequireTrue(t, fs.Write(core.JoinPath(dataDir, "auth.json"), "{}").OK)
}

func TestDispatchRuntime_ContainerCommandFor_OpencodeCreds_Good_Mounted(t *testing.T) {
	t.Setenv("AGENT_DOCKER_IMAGE", "")
	home := t.TempDir()
	t.Setenv("CORE_HOME", home) // HomeDir() reads CORE_HOME first
	// Host has an opencode credential → it mounts RO at the scratch path for an
	// opencode dispatch; the script copies it into a writable data dir.
	opencodeTestSeedCredential(t, home)

	script := opencodeAgentCommandScript("opencode-go/deepseek-v4-pro", "review")
	_, args := containerCommandFor(RuntimeDocker, "core-dev", false, "sh", []string{"-c", script}, "/ws", "/ws/.meta")
	joined := core.Join(" ", args...)

	core.AssertContains(t, joined, ":/run/oc-auth.json:ro")
	// The host's live data dir is NEVER bind-mounted — it holds a RW session DB.
	core.AssertNotContains(t, joined, "/home/agent/.local/share/opencode:")
}

func TestDispatchRuntime_ContainerCommandFor_OpencodeCreds_Bad_NoHostCredNoMount(t *testing.T) {
	t.Setenv("AGENT_DOCKER_IMAGE", "")
	home := t.TempDir() // no opencode credential on the host
	t.Setenv("CORE_HOME", home) // HomeDir() reads CORE_HOME first

	// An opencode dispatch on a host with no credential mounts nothing — the
	// free OpenCode Zen tier needs no auth. The script prelude still references
	// the scratch path harmlessly, so assert the absence of the MOUNT, not the
	// path text.
	script := opencodeAgentCommandScript("opencode/deepseek-v4-flash-free", "fix")
	_, args := containerCommandFor(RuntimeDocker, "core-dev", false, "sh", []string{"-c", script}, "/ws", "/ws/.meta")
	core.AssertNotContains(t, core.Join(" ", args...), ":/run/oc-auth.json:ro")
}

func TestDispatchRuntime_ContainerCommandFor_OpencodeCreds_Ugly_NonOpencodeNotMounted(t *testing.T) {
	t.Setenv("AGENT_DOCKER_IMAGE", "")
	home := t.TempDir()
	t.Setenv("CORE_HOME", home) // HomeDir() reads CORE_HOME first
	opencodeTestSeedCredential(t, home)

	// A codex dispatch does not reference the opencode scratch path, so the
	// credential is NOT exposed to it even though the host has one — the mount
	// is scoped to opencode dispatches, not all containers.
	_, args := containerCommandFor(RuntimeDocker, "core-dev", false, "codex", []string{"exec"}, "/ws", "/ws/.meta")
	core.AssertNotContains(t, core.Join(" ", args...), "oc-auth.json")
}

// --- dispatchRuntime / dispatchImage / dispatchGPU ---

func TestDispatchRuntime_DispatchRuntime_Good_Case(t *testing.T) {
	t.Setenv("CORE_AGENT_RUNTIME", "")
	c := core.New()
	c.Config().Set("agents.dispatch", DispatchConfig{Runtime: "podman"})
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{})}
	core.AssertEqual(t, "podman", s.dispatchRuntime())
}

func TestDispatchRuntime_DispatchRuntime_Bad_Case(t *testing.T) {
	t.Setenv("CORE_AGENT_RUNTIME", "")
	// Nil subsystem returns the auto default.
	var s *PrepSubsystem
	core.AssertEqual(t, RuntimeAuto, s.dispatchRuntime())
}

func TestDispatchRuntime_DispatchRuntime_Ugly_Case(t *testing.T) {
	// Env var override wins over configured runtime.
	t.Setenv("CORE_AGENT_RUNTIME", "apple")
	c := core.New()
	c.Config().Set("agents.dispatch", DispatchConfig{Runtime: "podman"})
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{})}
	core.AssertEqual(t, "apple", s.dispatchRuntime())
}

func TestDispatchRuntime_DispatchImage_Good_Case(t *testing.T) {
	t.Setenv("AGENT_DOCKER_IMAGE", "")
	c := core.New()
	c.Config().Set("agents.dispatch", DispatchConfig{Image: "core-ml"})
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{})}
	core.AssertEqual(t, "core-ml", s.dispatchImage())
}

func TestDispatchRuntime_DispatchImage_Bad_Case(t *testing.T) {
	t.Setenv("AGENT_DOCKER_IMAGE", "")
	// Nil subsystem falls back to the default image.
	var s *PrepSubsystem
	core.AssertEqual(t, defaultDockerImage, s.dispatchImage())
}

func TestDispatchRuntime_DispatchImage_Ugly_Case(t *testing.T) {
	// Env var override wins over configured image.
	t.Setenv("AGENT_DOCKER_IMAGE", "ad-hoc-image")
	c := core.New()
	c.Config().Set("agents.dispatch", DispatchConfig{Image: "core-ml"})
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{})}
	core.AssertEqual(t, "ad-hoc-image", s.dispatchImage())
}

func TestDispatchRuntime_DispatchGPU_Good_Case(t *testing.T) {
	c := core.New()
	c.Config().Set("agents.dispatch", DispatchConfig{GPU: true})
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{})}
	core.AssertTrue(t, s.dispatchGPU())
}

func TestDispatchRuntime_DispatchGPU_Bad_Case(t *testing.T) {
	// Nil subsystem returns false (GPU off by default).
	var s *PrepSubsystem
	enabled := s.dispatchGPU()
	core.AssertFalse(t, enabled)
	core.AssertEqual(t, false, enabled)
}

func TestDispatchRuntime_DispatchGPU_Ugly_Case(t *testing.T) {
	// Missing dispatch config returns false instead of panicking.
	c := core.New()
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{})}
	core.AssertFalse(t, s.dispatchGPU())
}

// isDarwin checks the host operating system without importing runtime in the
// test file (the import happens in dispatch.go where it's needed for the real
// detection logic).
func isDarwin() bool {
	return goosIsDarwin
}
