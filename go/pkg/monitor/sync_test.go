// SPDX-License-Identifier: EUPL-1.2

package monitor

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	core "dappco.re/go"
	"dappco.re/go/agent/pkg/messages"
)

func TestSync_InitSyncTimestamp_Good_Case(t *testing.T) {
	mon := New()
	mon.initSyncTimestamp()
	core.AssertTrue(t, mon.lastSyncTimestamp > 0)
}

func TestSync_InitSyncTimestamp_Bad_NoOverwrite(t *testing.T) {
	mon := New()
	mon.lastSyncTimestamp = 42
	mon.initSyncTimestamp()
	core.AssertEqual(t, int64(42), mon.lastSyncTimestamp)
}

func TestSync_SyncRepos_Ugly_NoBrainKey(t *testing.T) {
	t.Setenv("CORE_BRAIN_KEY", "")
	mon := New()
	mon.ServiceRuntime = testMon.ServiceRuntime
	result := mon.syncRepos()
	core.AssertEqual(t, "", result)
}

func TestSync_SyncRepos_Good_NoChanges(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/agent/checkin", r.URL.Path)
		resp := CheckinResponse{Timestamp: time.Now().Unix()}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(core.JSONMarshalString(resp)))
	}))
	defer srv.Close()

	setupAPIEnv(t, srv.URL)

	mon := New()
	msg := mon.syncRepos()
	core.AssertEqual(t, "", msg)
}

func TestSync_SyncRepos_Good_EncodesAgentQuery(t *testing.T) {
	expectedAgent := "test agent+1"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/agent/checkin", r.URL.Path)
		core.AssertEqual(t, expectedAgent, r.URL.Query().Get("agent"))
		resp := CheckinResponse{Timestamp: time.Now().Unix()}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(core.JSONMarshalString(resp)))
	}))
	defer srv.Close()

	setupAPIEnv(t, srv.URL)
	t.Setenv("AGENT_NAME", expectedAgent)

	mon := New()
	msg := mon.syncRepos()
	core.AssertEqual(t, "", msg)
}

func TestSync_SyncRepos_Bad_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	setupAPIEnv(t, srv.URL)

	mon := New()
	msg := mon.syncRepos()
	core.AssertEqual(t, "", msg)
}

func TestSync_SyncRepos_Good_UpdatesTimestamp(t *testing.T) {
	newTS := time.Now().Unix() + 1000
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := CheckinResponse{Timestamp: newTS}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(core.JSONMarshalString(resp)))
	}))
	defer srv.Close()

	setupAPIEnv(t, srv.URL)

	mon := New()
	mon.syncRepos()

	unlock := mon.monitorLock()
	core.AssertEqual(t, newTS, mon.lastSyncTimestamp)
	unlock()
}

func TestSync_SyncRepos_Good_PullsChangedRepo(t *testing.T) {
	remoteDir := core.JoinPath(t.TempDir(), "remote")
	fs.EnsureDir(remoteDir)
	run(t, remoteDir, "git", "init", "--bare")

	codeDir := t.TempDir()
	repoDir := core.JoinPath(codeDir, "test-repo")
	run(t, codeDir, "git", "clone", remoteDir, "test-repo")
	run(t, repoDir, "git", "checkout", "-b", "main")
	fs.Write(core.JoinPath(repoDir, "README.md"), "# test")
	run(t, repoDir, "git", "add", ".")
	run(t, repoDir, "git", "commit", "-m", "init")
	run(t, repoDir, "git", "push", "-u", "origin", "main")

	clone2Parent := t.TempDir()
	tmpClone := core.JoinPath(clone2Parent, "clone2")
	run(t, clone2Parent, "git", "clone", remoteDir, "clone2")
	run(t, tmpClone, "git", "checkout", "main")
	fs.Write(core.JoinPath(tmpClone, "new.go"), "package main\n")
	run(t, tmpClone, "git", "add", ".")
	run(t, tmpClone, "git", "commit", "-m", "agent work")
	run(t, tmpClone, "git", "push", "origin", "main")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := CheckinResponse{
			Changed:   []ChangedRepo{{Repo: "test-repo", Branch: "main", SHA: "abc"}},
			Timestamp: time.Now().Unix() + 100,
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(core.JSONMarshalString(resp)))
	}))
	defer srv.Close()

	setupAPIEnv(t, srv.URL)
	t.Setenv("CODE_PATH", codeDir)

	mon := New()
	mon.ServiceRuntime = testMon.ServiceRuntime
	msg := mon.syncRepos()
	core.AssertContains(t, msg, "Synced 1 repo(s)")
	core.AssertContains(t, msg, "test-repo")
}

func TestSync_HandleWorkspacePushed_Good_ResetsTrackedRepo(t *testing.T) {
	remoteDir := core.JoinPath(t.TempDir(), "remote")
	fs.EnsureDir(remoteDir)
	run(t, remoteDir, "git", "init", "--bare")

	codeDir := t.TempDir()
	orgDir := core.JoinPath(codeDir, "core")
	fs.EnsureDir(orgDir)
	repoDir := core.JoinPath(orgDir, "test-repo")
	run(t, orgDir, "git", "clone", remoteDir, "test-repo")
	run(t, repoDir, "git", "checkout", "-b", "main")
	fs.Write(core.JoinPath(repoDir, "README.md"), "# test")
	run(t, repoDir, "git", "add", ".")
	run(t, repoDir, "git", "commit", "-m", "init")
	run(t, repoDir, "git", "push", "-u", "origin", "main")

	cloneParent := t.TempDir()
	tmpClone := core.JoinPath(cloneParent, "clone2")
	run(t, cloneParent, "git", "clone", remoteDir, "clone2")
	run(t, tmpClone, "git", "checkout", "main")
	fs.Write(core.JoinPath(tmpClone, "new.go"), "package main\n")
	run(t, tmpClone, "git", "add", ".")
	run(t, tmpClone, "git", "commit", "-m", "agent work")
	run(t, tmpClone, "git", "push", "origin", "main")

	t.Setenv("CODE_PATH", codeDir)

	mon := New()
	mon.ServiceRuntime = testMon.ServiceRuntime

	result := mon.HandleIPCEvents(mon.Core(), messages.WorkspacePushed{
		Repo:   "test-repo",
		Branch: "main",
		Org:    "core",
	})
	core.AssertTrue(t, result.OK)
	core.AssertTrue(t, fs.Exists(core.JoinPath(repoDir, "new.go")).OK)
	core.AssertEqual(t, mon.gitOutput(tmpClone, "rev-parse", "HEAD"), mon.gitOutput(repoDir, "rev-parse", "HEAD"))
}

func TestSync_HandleWorkspacePushed_Good_SwitchesTrackedRepoBranch(t *testing.T) {
	remoteDir := core.JoinPath(t.TempDir(), "remote")
	fs.EnsureDir(remoteDir)
	run(t, remoteDir, "git", "init", "--bare")

	codeDir := t.TempDir()
	orgDir := core.JoinPath(codeDir, "core")
	fs.EnsureDir(orgDir)
	repoDir := core.JoinPath(orgDir, "test-repo")
	run(t, orgDir, "git", "clone", remoteDir, "test-repo")
	run(t, repoDir, "git", "checkout", "-b", "main")
	fs.Write(core.JoinPath(repoDir, "README.md"), "# test")
	run(t, repoDir, "git", "add", ".")
	run(t, repoDir, "git", "commit", "-m", "init")
	run(t, repoDir, "git", "push", "-u", "origin", "main")
	run(t, repoDir, "git", "checkout", "-b", "feature/wip")

	cloneParent := t.TempDir()
	tmpClone := core.JoinPath(cloneParent, "clone2")
	run(t, cloneParent, "git", "clone", remoteDir, "clone2")
	run(t, tmpClone, "git", "checkout", "main")
	fs.Write(core.JoinPath(tmpClone, "new.go"), "package main\n")
	run(t, tmpClone, "git", "add", ".")
	run(t, tmpClone, "git", "commit", "-m", "agent work")
	run(t, tmpClone, "git", "push", "origin", "main")

	t.Setenv("CODE_PATH", codeDir)

	mon := New()
	mon.ServiceRuntime = testMon.ServiceRuntime

	result := mon.HandleIPCEvents(mon.Core(), messages.WorkspacePushed{
		Repo:   "test-repo",
		Branch: "main",
		Org:    "core",
	})
	core.AssertTrue(t, result.OK)
	core.AssertEqual(t, "main", mon.gitOutput(repoDir, "rev-parse", "--abbrev-ref", "HEAD"))
	core.AssertTrue(t, fs.Exists(core.JoinPath(repoDir, "new.go")).OK)
	core.AssertEqual(t, mon.gitOutput(tmpClone, "rev-parse", "HEAD"), mon.gitOutput(repoDir, "rev-parse", "HEAD"))
}

func TestSync_SyncRepos_Good_NormalisesWindowsRepoPath(t *testing.T) {
	remoteDir := core.JoinPath(t.TempDir(), "remote")
	fs.EnsureDir(remoteDir)
	run(t, remoteDir, "git", "init", "--bare")

	codeDir := t.TempDir()
	repoDir := core.JoinPath(codeDir, "test-repo")
	run(t, codeDir, "git", "clone", remoteDir, "test-repo")
	run(t, repoDir, "git", "checkout", "-b", "main")
	fs.Write(core.JoinPath(repoDir, "README.md"), "# test")
	run(t, repoDir, "git", "add", ".")
	run(t, repoDir, "git", "commit", "-m", "init")
	run(t, repoDir, "git", "push", "-u", "origin", "main")

	clone2Parent := t.TempDir()
	tmpClone := core.JoinPath(clone2Parent, "clone2")
	run(t, clone2Parent, "git", "clone", remoteDir, "clone2")
	run(t, tmpClone, "git", "checkout", "main")
	fs.Write(core.JoinPath(tmpClone, "new.go"), "package main\n")
	run(t, tmpClone, "git", "add", ".")
	run(t, tmpClone, "git", "commit", "-m", "agent work")
	run(t, tmpClone, "git", "push", "origin", "main")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := CheckinResponse{
			Changed:   []ChangedRepo{{Repo: "core\\test-repo", Branch: "main", SHA: "abc"}},
			Timestamp: time.Now().Unix() + 100,
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(core.JSONMarshalString(resp)))
	}))
	defer srv.Close()

	setupAPIEnv(t, srv.URL)
	t.Setenv("CODE_PATH", codeDir)

	mon := New()
	mon.ServiceRuntime = testMon.ServiceRuntime
	msg := mon.syncRepos()
	core.AssertContains(t, msg, "Synced 1 repo(s)")
	core.AssertContains(t, msg, "core\\test-repo")
}

func TestSync_SyncRepos_Good_SkipsDirtyRepo(t *testing.T) {
	remoteDir := core.JoinPath(t.TempDir(), "remote")
	fs.EnsureDir(remoteDir)
	run(t, remoteDir, "git", "init", "--bare")

	codeDir := t.TempDir()
	repoDir := core.JoinPath(codeDir, "dirty-repo")
	run(t, codeDir, "git", "clone", remoteDir, "dirty-repo")
	run(t, repoDir, "git", "checkout", "-b", "main")
	fs.Write(core.JoinPath(repoDir, "README.md"), "# test")
	run(t, repoDir, "git", "add", ".")
	run(t, repoDir, "git", "commit", "-m", "init")
	run(t, repoDir, "git", "push", "-u", "origin", "main")

	fs.Write(core.JoinPath(repoDir, "dirty.txt"), "uncommitted")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := CheckinResponse{
			Changed:   []ChangedRepo{{Repo: "dirty-repo", Branch: "main", SHA: "abc"}},
			Timestamp: time.Now().Unix() + 100,
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(core.JSONMarshalString(resp)))
	}))
	defer srv.Close()

	setupAPIEnv(t, srv.URL)
	t.Setenv("CODE_PATH", codeDir)

	mon := New()
	mon.ServiceRuntime = testMon.ServiceRuntime
	msg := mon.syncRepos()
	core.AssertEqual(t, "", msg)
}

func TestSync_SyncRepos_Good_SkipsNonMainBranch(t *testing.T) {
	remoteDir := core.JoinPath(t.TempDir(), "remote")
	fs.EnsureDir(remoteDir)
	run(t, remoteDir, "git", "init", "--bare")

	codeDir := t.TempDir()
	repoDir := core.JoinPath(codeDir, "feature-repo")
	run(t, codeDir, "git", "clone", remoteDir, "feature-repo")
	run(t, repoDir, "git", "checkout", "-b", "main")
	fs.Write(core.JoinPath(repoDir, "README.md"), "# test")
	run(t, repoDir, "git", "add", ".")
	run(t, repoDir, "git", "commit", "-m", "init")
	run(t, repoDir, "git", "push", "-u", "origin", "main")
	run(t, repoDir, "git", "checkout", "-b", "feature/wip")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := CheckinResponse{
			Changed:   []ChangedRepo{{Repo: "feature-repo", Branch: "main", SHA: "abc"}},
			Timestamp: time.Now().Unix() + 100,
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(core.JSONMarshalString(resp)))
	}))
	defer srv.Close()

	setupAPIEnv(t, srv.URL)
	t.Setenv("CODE_PATH", codeDir)

	mon := New()
	mon.ServiceRuntime = testMon.ServiceRuntime
	msg := mon.syncRepos()
	core.AssertEqual(t, "", msg)
}

func TestSync_SyncRepos_Good_SkipsNonexistentRepo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := CheckinResponse{
			Changed:   []ChangedRepo{{Repo: "nonexistent", Branch: "main", SHA: "abc"}},
			Timestamp: time.Now().Unix() + 100,
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(core.JSONMarshalString(resp)))
	}))
	defer srv.Close()

	setupAPIEnv(t, srv.URL)
	t.Setenv("CODE_PATH", t.TempDir())

	mon := New()
	msg := mon.syncRepos()
	core.AssertEqual(t, "", msg)
}

func TestSync_SyncRepos_Good_UsesEnvBrainKey(t *testing.T) {
	var authHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		resp := CheckinResponse{Timestamp: time.Now().Unix()}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(core.JSONMarshalString(resp)))
	}))
	defer srv.Close()

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CORE_BRAIN_KEY", "env-key-value")
	t.Setenv("CORE_API_URL", srv.URL)
	t.Setenv("AGENT_NAME", "test-agent")

	mon := New()
	mon.syncRepos()
	core.AssertEqual(t, "Bearer env-key-value", authHeader)
}
