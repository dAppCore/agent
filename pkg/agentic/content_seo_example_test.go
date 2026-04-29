// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"syscall"
	"time"

	core "dappco.re/go"
	"github.com/gin-gonic/gin"
)

func contentSEOExampleSubsystem(name string) (*PrepSubsystem, func()) {
	fsys := (&core.Fs{}).NewUnrestricted()
	root := core.JoinPath("/tmp", name)
	_ = fsys.DeleteAll(root)

	oldWorkspace := core.Getenv("CORE_WORKSPACE")
	if err := syscall.Setenv("CORE_WORKSPACE", root); err != nil {
		panic(err)
	}

	subsystem := &PrepSubsystem{}
	cleanup := func() {
		subsystem.closeStateStore()
		if oldWorkspace == "" {
			if err := syscall.Unsetenv("CORE_WORKSPACE"); err != nil {
				panic(err)
			}
		} else if err := syscall.Setenv("CORE_WORKSPACE", oldWorkspace); err != nil {
			panic(err)
		}
		_ = fsys.DeleteAll(root)
	}
	return subsystem, cleanup
}

func ExamplePrepSubsystem_ScheduleRevision() {
	subsystem, cleanup := contentSEOExampleSubsystem("core-agent-content-seo-schedule")
	defer cleanup()

	originalNow := contentSEONow
	contentSEONow = func() time.Time {
		return time.Date(2026, time.April, 26, 12, 0, 0, 0, time.UTC)
	}
	defer func() { contentSEONow = originalNow }()

	revision, err := subsystem.ScheduleRevision(context.Background(), "/help/hosting", "Updated copy")
	core.Println(err == nil)
	core.Println(revision.PageID)
	core.Println(revision.ScheduledAt == nil)
	// Output:
	// true
	// /help/hosting
	// true
}

func ExamplePrepSubsystem_GetPendingRevisions() {
	subsystem, cleanup := contentSEOExampleSubsystem("core-agent-content-seo-pending")
	defer cleanup()

	_, _ = subsystem.ScheduleRevision(context.Background(), "/help/hosting", "Updated copy")
	revisions, err := subsystem.GetPendingRevisions("/help/hosting")
	core.Println(err == nil)
	core.Println(len(revisions))
	// Output:
	// true
	// 1
}

func ExamplePrepSubsystem_OnGooglebotVisit() {
	subsystem, cleanup := contentSEOExampleSubsystem("core-agent-content-seo-visit")
	defer cleanup()

	originalNow := contentSEONow
	originalDelay := contentSEORandomDelay
	contentSEONow = func() time.Time {
		return time.Date(2026, time.April, 26, 12, 0, 0, 0, time.UTC)
	}
	contentSEORandomDelay = func() (time.Duration, error) { return 37 * time.Minute, nil }
	defer func() {
		contentSEONow = originalNow
		contentSEORandomDelay = originalDelay
	}()

	_, _ = subsystem.ScheduleRevision(context.Background(), "/help/hosting", "Updated copy")
	err := subsystem.OnGooglebotVisit(context.Background(), "/help/hosting")
	pending, _ := subsystem.GetPendingRevisions("/help/hosting")
	core.Println(err == nil)
	core.Println(len(pending))
	// Output:
	// true
	// 0
}

func ExamplePrepSubsystem_HandleGooglebotVisit() {
	subsystem, cleanup := contentSEOExampleSubsystem("core-agent-content-seo-handle")
	defer cleanup()

	originalNow := contentSEONow
	originalDelay := contentSEORandomDelay
	contentSEONow = func() time.Time {
		return time.Date(2026, time.April, 26, 12, 0, 0, 0, time.UTC)
	}
	contentSEORandomDelay = func() (time.Duration, error) { return 37 * time.Minute, nil }
	defer func() {
		contentSEONow = originalNow
		contentSEORandomDelay = originalDelay
	}()

	_, _ = subsystem.ScheduleRevision(context.Background(), "/help/hosting", "Updated copy")
	err := subsystem.HandleGooglebotVisit(context.Background(), "/help/hosting", "Googlebot/2.1")
	pending, _ := subsystem.GetPendingRevisions("/help/hosting")
	core.Println(err == nil)
	core.Println(len(pending))
	// Output:
	// true
	// 0
}

func ExamplePrepSubsystem_ContentSEOGooglebotMiddleware() {
	subsystem, cleanup := contentSEOExampleSubsystem("core-agent-content-seo-middleware")
	defer cleanup()

	originalNow := contentSEONow
	originalDelay := contentSEORandomDelay
	contentSEONow = func() time.Time {
		return time.Date(2026, time.April, 26, 12, 0, 0, 0, time.UTC)
	}
	contentSEORandomDelay = func() (time.Duration, error) { return 37 * time.Minute, nil }
	defer func() {
		contentSEONow = originalNow
		contentSEORandomDelay = originalDelay
	}()

	_, _ = subsystem.ScheduleRevision(context.Background(), "/help/hosting", "Updated copy")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/help/hosting", nil)
	c.Request.Header.Set("User-Agent", "Googlebot/2.1")

	subsystem.ContentSEOGooglebotMiddleware(nil)(c)

	pending, _ := subsystem.GetPendingRevisions("/help/hosting")
	core.Println(len(pending))
	// Output: 0
}
