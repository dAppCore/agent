// SPDX-Licence-Identifier: EUPL-1.2

// Migrated-store coverage: mount an in-memory DuckDB medium with the
// Imported* + Sandbox tables registered, so the ORM round-trips. This
// flips persistProjects/persistProviders from body-only (Save fails,
// count 0) to a real persist + read-back, and exercises the control
// read handlers on their SUCCESS path (200 + populated JSON) rather than
// just the unbacked-store failure branch.
//
// No docker/process: every call here is ORM I/O against a temp DuckDB.

package opencode

import (
	"net/http"
	"net/http/httptest"
	"testing"

	core "dappco.re/go"
	"dappco.re/go/orm"
	"github.com/gin-gonic/gin"
)

// newBackedService is newTestService + a mounted in-memory DuckDB medium
// with the opencode tables registered, so orm.Of[T](c).Save/Get round-trip.
func newBackedService(t *testing.T) *Service {
	t.Helper()
	svc := newTestService(t)

	mr := orm.NewDuckDB(":memory:")
	core.AssertTrue(t, mr.OK)
	m, _ := mr.Value.(*orm.DuckDBMedium)
	t.Cleanup(func() { _ = m.Close() })

	m.RegisterTable("imported_projects", ImportedProject{}.Schema())
	m.RegisterTable("imported_providers", ImportedProvider{}.Schema())
	m.RegisterTable("opencode_sandboxes", Sandbox{}.Schema())

	core.AssertTrue(t, orm.Mount(svc.Core(), "default", m).OK)
	return svc
}

// TestImportStore_persistProjects_RealCount — with a backed store the
// well-formed rows actually persist and the returned count is the number
// of saved projects (2 here: the full row + the virtual-worktree row;
// the non-map and id-less entries are skipped). The rows read back via
// ListImports.
func TestImportStore_persistProjects_RealCount(t *testing.T) {
	svc := newBackedService(t)
	now := core.Now()

	projects := []any{
		map[string]any{
			"id":       "proj-1",
			"worktree": "/home/user/myrepo",
			"vcs":      "git",
			"icon":     map[string]any{"color": "#abc", "url": "data:image/png;base64,AAAA"},
			"time":     map[string]any{"created": float64(1700000000000), "updated": float64(1700000100000)},
		},
		map[string]any{"id": "proj-2", "worktree": "/"},
		"not-a-map",
		map[string]any{"worktree": "/tmp/no-id"},
	}

	got := persistProjects(svc.Core(), projects, now)
	core.AssertEqual(t, 2, got)

	// Read-back through the Service — success path of ListImports.
	r := svc.ListImports()
	core.AssertTrue(t, r.OK)
	rows, _ := r.Value.([]ImportedProject)
	core.AssertEqual(t, 2, len(rows))
}

// TestImportStore_persistProviders_RealCount — backed store: 2 providers
// persist, one of them with auth (anthropic via the authMap). The
// (count, withAuth) return is (2, 1); rows read back via
// ListImportedProviders.
func TestImportStore_persistProviders_RealCount(t *testing.T) {
	svc := newBackedService(t)
	now := core.Now()

	providers := []any{
		map[string]any{"id": "anthropic", "name": "Anthropic", "npm": "@ai-sdk/anthropic",
			"options": map[string]any{"baseURL": "https://api.anthropic.com"}},
		map[string]any{"id": "openai", "name": "OpenAI"},
		42,
		map[string]any{"name": "no-id"},
	}
	authMap := map[string]map[string]any{
		"anthropic": {"type": "apikey", "key": "sk-secret"},
	}

	count, withAuth := persistProviders(svc.Core(), providers, authMap, now)
	core.AssertEqual(t, 2, count)
	core.AssertEqual(t, 1, withAuth)

	r := svc.ListImportedProviders()
	core.AssertTrue(t, r.OK)
	rows, _ := r.Value.([]ImportedProvider)
	core.AssertEqual(t, 2, len(rows))
}

// TestControl_ReadHandlers_SuccessPaths_HTTP — with a backed store the
// imports/list/inspect handlers return 200 with populated bodies. This
// covers the success leg of listImports/listImportedProviders/list/inspect
// that the unbacked-store test could only reach on its error branch.
func TestControl_ReadHandlers_SuccessPaths_HTTP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := newBackedService(t)

	// Seed one project, one provider, one sandbox so each list/inspect has
	// a row to return.
	persistProjects(svc.Core(), []any{map[string]any{"id": "p1", "worktree": "/home/u/r"}}, core.Now())
	persistProviders(svc.Core(), []any{map[string]any{"id": "anthropic", "name": "Anthropic"}}, nil, core.Now())
	sb := Sandbox{ID: "oc-seed", Image: "img", HostPort: 51823, Status: StatusRunning, CreatedAt: core.Now()}
	core.AssertTrue(t, orm.Of[Sandbox](svc.Core()).Save(&sb).OK)

	g := NewControlGroup(svc)
	e := gin.New()
	g.RegisterRoutes(e.Group(""))

	do := func(method, path string) (int, string) {
		w := httptest.NewRecorder()
		e.ServeHTTP(w, httptest.NewRequest(method, path, nil))
		return w.Code, w.Body.String()
	}

	code, body := do("GET", "/imports")
	core.AssertEqual(t, http.StatusOK, code)
	core.AssertContains(t, body, "p1")

	code, body = do("GET", "/imports/providers")
	core.AssertEqual(t, http.StatusOK, code)
	core.AssertContains(t, body, "anthropic")

	code, body = do("GET", "/sandbox")
	core.AssertEqual(t, http.StatusOK, code)
	core.AssertContains(t, body, "oc-seed")

	code, body = do("GET", "/sandbox/oc-seed")
	core.AssertEqual(t, http.StatusOK, code)
	core.AssertContains(t, body, "oc-seed")
}
