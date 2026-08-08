// SPDX-License-Identifier: EUPL-1.2

package main

import (
	"bufio"
	"context"

	core "dappco.re/go"
	"dappco.re/go/agent/pkg/chathistory"
	"dappco.re/go/agent/pkg/lemma"
)

// chat is the user-facing REPL command. Opens (or creates) the user's
// chathistory archive at ~/Lethean/data/users/<user>/chats.duckdb,
// starts a Lemma session against the configured lthn-mlx serve
// endpoint, and pipes stdin lines through Send(). Every turn captures
// to the archive automatically — see project_chat_continuity_rights_
// normal_user_pattern for the why.
//
//	core-agent chat --user=owlet
//	core-agent chat --user=owlet --title="evening vent"
//	core-agent chat --user=owlet --base-url=http://tunnel:11434/v1 --model=gemma-4-27b-bf16
//	core-agent chat --user=owlet --workdir=/tmp/owlet-test.duckdb
//
// REPL commands inside the loop:
//
//	/quit   end session, close conversation, exit
//	/exit   same as /quit
func (commands applicationCommandSet) chat(opts core.Options) core.Result {
	user := opts.String("user")
	if user == "" {
		applicationPrint("chat: --user=<id> is required")
		return core.Result{}
	}

	workdir := opts.String("workdir")
	if workdir == "" {
		workdir = defaultUserChatsPath(user)
	}
	baseURL := opts.String("base-url")
	if baseURL == "" {
		baseURL = lemma.DefaultBaseURL
	}
	modelID := opts.String("model")
	if modelID == "" {
		modelID = lemma.DefaultModelID
	}
	title := opts.String("title")

	hist, err := chathistory.Open(user, workdir)
	if err != nil {
		applicationPrint("chat: open archive: %v", err)
		return core.Result{}
	}
	defer func() {
		// Reported, not dropped: this handle is written to, so a failed
		// close can mean the last turns never reached the archive.
		if err := hist.Close(); err != nil {
			core.Warn("chat: failed to close history archive", "reason", err)
		}
	}()

	svc := lemma.New(lemma.Config{
		BaseURL: baseURL,
		ModelID: modelID,
		History: hist,
	})
	sess, err := svc.StartSession(user, lemma.SessionMeta{Title: title})
	if err != nil {
		applicationPrint("chat: start session: %v", err)
		return core.Result{}
	}
	defer func() { _ = sess.End() }()

	applicationPrint("core-agent chat — user=%s model=%s", user, modelID)
	applicationPrint("  endpoint:     %s", baseURL)
	applicationPrint("  archive:      %s", workdir)
	applicationPrint("  conversation: %s", sess.ConversationID())
	applicationPrint("type /quit to end (ctrl-d / ctrl-c also work)")
	applicationPrint("")

	stdout := core.Stdout()
	scanner := bufio.NewScanner(core.Stdin())
	scanner.Buffer(make([]byte, 64*1024), 1024*1024) // allow long prompts
	for {
		// Prompt only: a failed terminal write is cosmetic and there is
		// nowhere to return an error to inside the loop.
		_ = core.WriteString(stdout, "you: ")
		if !scanner.Scan() {
			break
		}
		line := core.Trim(scanner.Text())
		if line == "" {
			continue
		}
		if line == "/quit" || line == "/exit" {
			break
		}
		reply, err := sess.Send(context.Background(), line)
		if err != nil {
			applicationPrint("error: %v", err)
			continue
		}
		applicationPrint("lemma: %s", reply)
		applicationPrint("")
	}

	applicationPrint("")
	applicationPrint("conversation saved to %s", workdir)
	return core.Result{OK: true}
}

// defaultUserChatsPath returns ~/Lethean/data/users/<user>/chats.duckdb,
// matching the convention chathistory and the agent's data tree expect.
func defaultUserChatsPath(user string) string {
	return core.PathJoin(core.Env("HOME"), "Lethean", "data", "users", user, "chats.duckdb")
}
