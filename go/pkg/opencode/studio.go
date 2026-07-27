// SPDX-Licence-Identifier: EUPL-1.2

// Studio — host-side OpenCode native app detection + launch.
//
// Per RFC.opencode.md §6 "Open Studio (host-app)". Optional path —
// users who already run OpenCode's native desktop app on the host
// and want T1 config only get a one-click launcher here. The
// button is hidden when the app isn't detected.
//
// Platform paths:
//
//   - darwin → /Applications/OpenCode.app present? `open -a OpenCode`
//   - linux  → `opencode-studio` or similar binary on PATH (TBD —
//     opencode doesn't ship a Linux desktop app today; placeholder).
//   - windows → %ProgramFiles%/OpenCode/opencode.exe (TBD).
//
// IsStudioInstalled is the gate the frontend uses to decide
// whether to render the button at all.

package opencode

import (
	"context"
	goruntime "runtime"

	core "dappco.re/go"
)

// studioMacPath is the canonical install location for OpenCode's
// macOS desktop app. Other paths (Setapp / sideloaded) aren't
// detected today — users can still launch via Spotlight.
const studioMacPath = "/Applications/OpenCode.app"

// studioInstalled reports whether OpenCode's native desktop app is
// present on the host. Indirected through a package var — mirroring
// portProbe / pickPortInRange in opencode.go — so the openStudio
// control handler can drive both its present (200) and absent (404)
// legs without depending on what is actually installed on the test
// host. The default does the real per-platform filesystem / PATH
// detection.
//
// Usage example:
//
//	orig := studioInstalled
//	defer func() { studioInstalled = orig }()
//	studioInstalled = func() bool { return true }
var studioInstalled = func() bool {
	switch goruntime.GOOS {
	case "darwin":
		return core.Stat(studioMacPath).OK
	case "linux":
		// opencode doesn't ship a Linux desktop app today — leaving
		// the hook in place for when they do.
		return false
	case "windows":
		// Same — Windows desktop app TBD upstream.
		return false
	default:
		return false
	}
}

// studioOpen launches the detected native app. Indirected through a
// package var so openStudio's present leg can assert a 200 without
// shelling a real `open -a OpenCode` (which would launch a GUI on the
// test host, or fail when the app isn't installed). The default does
// the real launch via the process service.
//
// Usage example:
//
//	orig := studioOpen
//	defer func() { studioOpen = orig }()
//	studioOpen = func(*Service, context.Context) core.Result { return core.Ok(true) }
var studioOpen = func(s *Service, ctx context.Context) core.Result {
	ps := s.proc()
	if ps == nil {
		return core.Fail(core.E("opencode.OpenStudio", "process service unavailable", nil))
	}
	switch goruntime.GOOS {
	case "darwin":
		return ps.Run(ctx, "open", "-a", "OpenCode")
	default:
		return core.Fail(core.E("opencode.OpenStudio",
			"unsupported platform: "+goruntime.GOOS, nil))
	}
}

// IsStudioInstalled reports whether OpenCode's native desktop app
// is installed on the host. Frontend uses this to decide whether
// to render the "Open Studio" button on the integrations card.
//
// Usage example:
//
//	if svc.IsStudioInstalled() { /* render the button */ }
func (s *Service) IsStudioInstalled() bool {
	return studioInstalled()
}

// OpenStudio launches the host's OpenCode native app. Returns
// Fail when the app isn't installed or the launch command errors.
//
// Usage example:
//
//	r := svc.OpenStudio()
//	if !r.OK { core.Println("open studio failed:", r.Error()) }
func (s *Service) OpenStudio() core.Result {
	if s == nil {
		return core.Fail(core.E("opencode.OpenStudio", "service is nil", nil))
	}
	if !s.IsStudioInstalled() {
		return core.Fail(core.E("opencode.OpenStudio",
			"OpenCode native app is not installed on this host", nil))
	}

	ctx, cancel := core.WithTimeout(core.Background(), 10*core.Second)
	defer cancel()

	return studioOpen(s, ctx)
}
