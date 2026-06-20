// SPDX-Licence-Identifier: EUPL-1.2

package opencode

import (
	"testing"

	core "dappco.re/go"
)

// TestWails_Lifecycle — the Wails service-lifecycle hooks + namespace
// label are pure and always succeed.
func TestWails_Lifecycle(t *testing.T) {
	w := NewWailsService(nil)
	core.AssertEqual(t, "OpenCodeWails", w.ServiceName())
	core.AssertTrue(t, w.ServiceStartup(core.Background(), nil).OK)
	core.AssertTrue(t, w.ServiceShutdown().OK)
}

// TestWails_NilService_Guards — every binding method must fail closed
// (or return the documented Ok(false) shape) when the embedded Service
// is unbound, rather than panicking. This is the renderer-boundary
// contract: an unbound binding surfaces "service not bound", never a
// nil-pointer crash that takes the WebView down.
func TestWails_NilService_Guards(t *testing.T) {
	w := &WailsService{svc: nil}

	// Methods that fail closed (Result.OK == false).
	failClosed := map[string]core.Result{
		"WStart":                 w.WStart("default"),
		"WStop":                  w.WStop("oc-1"),
		"WStatus":                w.WStatus(),
		"WInspect":               w.WInspect("oc-1"),
		"WListProfiles":          w.WListProfiles(),
		"WGetProfile":            w.WGetProfile("default"),
		"WSaveProfile":           w.WSaveProfile(Profile{}),
		"WDeleteProfile":         w.WDeleteProfile("x"),
		"WWebURL":                w.WWebURL("oc-1"),
		"WOpenWebWindow":         w.WOpenWebWindow("oc-1"),
		"WImportFromHost":        w.WImportFromHost(),
		"WListImports":           w.WListImports(),
		"WListImportedProviders": w.WListImportedProviders(),
		"WUpgradeWithConsent":    w.WUpgradeWithConsent(UpgradeInput{}),
		"WOpenStudio":            w.WOpenStudio(),
		"WOpenTUI":               w.WOpenTUI("oc-1"),
		"WEnable":                w.WEnable("default"),
		"WDisable":               w.WDisable(),
		"WProviderList":          w.WProviderList("oc-1"),
		"WMergeHostConfig":       w.WMergeHostConfig(MergeHostConfigOptions{}),
	}
	for name, r := range failClosed {
		if r.OK {
			t.Errorf("%s on unbound WailsService should be !OK", name)
		}
	}

	// Methods that return Ok(false) by design (UI render-hint queries
	// must not error just because the service is unbound).
	if r := w.WIsStudioInstalled(); !r.OK || r.Value != false {
		t.Errorf("WIsStudioInstalled unbound = (OK=%v,val=%v); want (true,false)", r.OK, r.Value)
	}
	if r := w.WIsEnabled(); !r.OK || r.Value != false {
		t.Errorf("WIsEnabled unbound = (OK=%v,val=%v); want (true,false)", r.OK, r.Value)
	}
}

// TestWails_NilReceiver_Guards — the methods that check `w == nil`
// (not just w.svc) must also survive a nil *WailsService receiver.
func TestWails_NilReceiver_Guards(t *testing.T) {
	var w *WailsService
	core.AssertFalse(t, w.WStart("").OK)
	core.AssertFalse(t, w.WStop("oc-1").OK)
	core.AssertFalse(t, w.WStatus().OK)
	core.AssertFalse(t, w.WInspect("oc-1").OK)
	core.AssertFalse(t, w.WListProfiles().OK)
	core.AssertFalse(t, w.WListImportedProviders().OK)
	core.AssertFalse(t, w.WMergeHostConfig(MergeHostConfigOptions{}).OK)
	// Ok(false)-shaped queries survive a nil receiver too.
	core.AssertTrue(t, w.WIsStudioInstalled().OK)
	core.AssertTrue(t, w.WIsEnabled().OK)
}

// TestWails_BoundService_SafeDelegators — with a real (but storeless +
// process-less) Service the binding delegators reach their Service
// method and surface its clean failure / read result. None of these
// paths touch docker or spawn a process: Status/Inspect are ORM reads,
// WebURL/ProviderList/OpenTUI fail at the not-running guard, Disable
// is a no-op stop-sweep, the upgrade gate fails closed without a
// confirmation, and the import/provider reads fail without a migrated
// store.
func TestWails_BoundService_SafeDelegators(t *testing.T) {
	svc := newTestService(t)
	w := NewWailsService(svc)

	// ORM reads against an unmigrated Sandbox table fail cleanly.
	core.AssertFalse(t, w.WStatus().OK)
	core.AssertFalse(t, w.WInspect("oc-1").OK)

	// Profile reads ARE migrated (seeded default) — these succeed.
	core.AssertTrue(t, w.WListProfiles().OK)
	core.AssertTrue(t, w.WGetProfile("default").OK)

	// Not-running / unavailable guards — never reach docker.
	core.AssertFalse(t, w.WWebURL("oc-1").OK)
	core.AssertFalse(t, w.WProviderList("oc-1").OK)
	core.AssertFalse(t, w.WOpenTUI("oc-1").OK)
	core.AssertFalse(t, w.WListImports().OK)
	core.AssertFalse(t, w.WListImportedProviders().OK)

	// Disable's stop-sweep is a no-op when nothing is running → OK.
	core.AssertTrue(t, w.WDisable().OK)

	// IsEnabled / IsStudioInstalled queries return a bool Result.
	core.AssertTrue(t, w.WIsEnabled().OK)
	core.AssertTrue(t, w.WIsStudioInstalled().OK)

	// Upgrade consent-gate fails closed without ConfirmedByUser — no
	// network call, no image pull (Cerberus #22 MED-2 / Mantis #1619).
	core.AssertFalse(t, w.WUpgradeWithConsent(UpgradeInput{}).OK)
}
