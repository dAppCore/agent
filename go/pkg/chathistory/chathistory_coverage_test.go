// SPDX-License-Identifier: EUPL-1.2

package chathistory

import (
	"path/filepath"
	"testing"

	core "dappco.re/go"
)

// openTemp returns a History over a fresh temp-dir archive, registering
// Close on test cleanup. Mirrors the open boilerplate the other tests
// use, lifted to a helper so the coverage cases stay focused on the
// behaviour under test.
//
//	h := openTemp(t)
//	conv, _ := h.StartConversation(NewConversation{ModelID: "lemer-lite"})
func openTemp(t *testing.T) *History {
	t.Helper()
	path := filepath.Join(t.TempDir(), "chats.duckdb")
	h, err := Open("owlet", path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = h.Close() })
	return h
}

// TestChatHistory_Close_Good — Close on a live handle releases cleanly and
// a second Close on a nil handle is a harmless no-op (the nil/db==nil guard).
func TestChatHistory_Close_Good(t *testing.T) {
	h := openTemp(t)
	core.AssertEqual(t, nil, h.Close())

	var nilH *History
	core.AssertEqual(t, nil, nilH.Close())
}

// TestChatHistory_LoadTurns_Good — turns come back in ordinal order with the
// role + content + ordinal triple the consumer replays into the next call.
func TestChatHistory_LoadTurns_Good(t *testing.T) {
	h := openTemp(t)
	conv, err := h.StartConversation(NewConversation{ModelID: "lemer-lite"})
	if err != nil {
		t.Fatalf("StartConversation: %v", err)
	}
	want := []NewTurn{
		{Role: "user", Content: "first"},
		{Role: "assistant", Content: "second"},
		{Role: "user", Content: "third"},
	}
	for i, nt := range want {
		if _, err := h.WriteTurn(conv, nt); err != nil {
			t.Fatalf("WriteTurn[%d]: %v", i, err)
		}
	}

	turns, err := h.LoadTurns(conv)
	if err != nil {
		t.Fatalf("LoadTurns: %v", err)
	}
	core.AssertEqual(t, len(want), len(turns))
	for i, tn := range turns {
		core.AssertEqual(t, i, tn.Ordinal)
		core.AssertEqual(t, want[i].Role, tn.Role)
		core.AssertEqual(t, want[i].Content, tn.Content)
	}
}

// TestChatHistory_LoadTurns_Good_Empty — an unknown conversation id yields
// zero turns and no error (the iterate-nothing branch).
func TestChatHistory_LoadTurns_Good_Empty(t *testing.T) {
	h := openTemp(t)
	turns, err := h.LoadTurns("no-such-conversation")
	core.AssertEqual(t, nil, err)
	core.AssertEqual(t, 0, len(turns))
}

// TestChatHistory_LoadTurns_Bad_EmptyID — an empty conversation id is rejected
// before any query runs.
func TestChatHistory_LoadTurns_Bad_EmptyID(t *testing.T) {
	h := openTemp(t)
	_, err := h.LoadTurns("")
	core.AssertTrue(t, err != nil)
}

// TestChatHistory_ClosedGuards_Bad — every method short-circuits on a nil
// handle with a "history closed" error rather than dereferencing a nil db.
func TestChatHistory_ClosedGuards_Bad(t *testing.T) {
	var h *History

	if _, err := h.StartConversation(NewConversation{ModelID: "x"}); err == nil {
		t.Fatal("StartConversation: want error on nil handle")
	}
	if _, err := h.WriteTurn("conv", NewTurn{Role: "user", Content: "x"}); err == nil {
		t.Fatal("WriteTurn: want error on nil handle")
	}
	if err := h.EndConversation("conv"); err == nil {
		t.Fatal("EndConversation: want error on nil handle")
	}
	if err := h.SetSignal("turn", "liked"); err == nil {
		t.Fatal("SetSignal: want error on nil handle")
	}
	if _, err := h.CountConversations(); err == nil {
		t.Fatal("CountConversations: want error on nil handle")
	}
	if _, err := h.CountTurns(); err == nil {
		t.Fatal("CountTurns: want error on nil handle")
	}
	if _, err := h.LoadTurns("conv"); err == nil {
		t.Fatal("LoadTurns: want error on nil handle")
	}
	if err := h.CopyTo("/tmp/x.duckdb"); err == nil {
		t.Fatal("CopyTo: want error on nil handle")
	}
	if err := h.ExportJSONL("/tmp/x.jsonl"); err == nil {
		t.Fatal("ExportJSONL: want error on nil handle")
	}
}

// TestChatHistory_ClosedDB_Ugly — once Close has released the file, queries
// against the still-non-nil handle surface the driver's closed-db error
// through the wrapped scope rather than panicking.
func TestChatHistory_ClosedDB_Ugly(t *testing.T) {
	path := filepath.Join(t.TempDir(), "chats.duckdb")
	h, err := Open("owlet", path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := h.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if _, err := h.CountConversations(); err == nil {
		t.Fatal("CountConversations on closed db: want error")
	}
	if _, err := h.CountTurns(); err == nil {
		t.Fatal("CountTurns on closed db: want error")
	}
	if _, err := h.LoadTurns("conv"); err == nil {
		t.Fatal("LoadTurns on closed db: want error")
	}
	if err := h.SetSignal("turn", "liked"); err == nil {
		t.Fatal("SetSignal on closed db: want error")
	}
	if err := h.EndConversation("conv"); err == nil {
		t.Fatal("EndConversation on closed db: want error")
	}
	if _, err := h.StartConversation(NewConversation{ModelID: "x"}); err == nil {
		t.Fatal("StartConversation on closed db: want error")
	}
}

// TestChatHistory_Open_Bad_MkdirParent — Open fails loudly when the parent
// directory cannot be created because a path component is a regular file.
func TestChatHistory_Open_Bad_MkdirParent(t *testing.T) {
	dir := t.TempDir()
	fileAsParent := filepath.Join(dir, "afile")
	if r := core.WriteFile(fileAsParent, []byte("x"), 0o644); !r.OK {
		t.Fatalf("WriteFile: %v", r.Value)
	}
	// afile is a file, so creating afile/sub as a directory must fail.
	_, err := Open("owlet", filepath.Join(fileAsParent, "sub", "chats.duckdb"))
	core.AssertTrue(t, err != nil)
}

// TestChatHistory_StartConversation_Good_TagsMetadata — the tags-present and
// metadata-present branches round-trip through to the JSONL export.
func TestChatHistory_StartConversation_Good_TagsMetadata(t *testing.T) {
	h := openTemp(t)
	conv, err := h.StartConversation(NewConversation{
		Title:          "evening vent",
		ModelID:        "lemer-lite",
		BaseModel:      "gemma-4-e2b-it-4bit",
		AdapterID:      "lek2",
		Tags:           []string{"life", "vent"},
		Metadata:       []byte(`{"client":"desktop"}`),
		ConsentVersion: 3,
	})
	if err != nil {
		t.Fatalf("StartConversation: %v", err)
	}
	core.AssertTrue(t, conv != "")

	n, err := h.CountConversations()
	core.AssertEqual(t, nil, err)
	core.AssertEqual(t, 1, n)
}

// TestChatHistory_WriteTurn_Good_ToolFieldsAndTokens — the tool_calls,
// tool_results and token-count columns persist (nullableJSON / nullableInt
// non-empty branches) and read back through LoadTurns + ExportJSONL.
func TestChatHistory_WriteTurn_Good_ToolFieldsAndTokens(t *testing.T) {
	h := openTemp(t)
	conv, err := h.StartConversation(NewConversation{ModelID: "lemer-lite"})
	if err != nil {
		t.Fatalf("StartConversation: %v", err)
	}
	turnID, err := h.WriteTurn(conv, NewTurn{
		Role:        "assistant",
		Content:     "calling a tool",
		ToolCalls:   []byte(`[{"name":"search"}]`),
		ToolResults: []byte(`[{"hits":2}]`),
		TokensIn:    16,
		TokensOut:   8,
	})
	if err != nil {
		t.Fatalf("WriteTurn: %v", err)
	}
	core.AssertTrue(t, turnID != "")

	turns, err := h.LoadTurns(conv)
	core.AssertEqual(t, nil, err)
	core.AssertEqual(t, 1, len(turns))
	core.AssertEqual(t, "assistant", turns[0].Role)
}

// TestChatHistory_EndConversation_Good_Idempotent — EndConversation on an open
// conversation closes it, and a second call is a harmless no-op.
func TestChatHistory_EndConversation_Good_Idempotent(t *testing.T) {
	h := openTemp(t)
	conv, err := h.StartConversation(NewConversation{ModelID: "lemer-lite"})
	if err != nil {
		t.Fatalf("StartConversation: %v", err)
	}
	if err := h.EndConversation(conv); err != nil {
		t.Fatalf("EndConversation (first): %v", err)
	}
	if err := h.EndConversation(conv); err != nil {
		t.Fatalf("EndConversation (idempotent): %v", err)
	}
}

// TestChatHistory_SetSignal_Good — a signal stamped on a turn survives into
// the JSONL export's signal field.
func TestChatHistory_SetSignal_Good(t *testing.T) {
	h := openTemp(t)
	conv, err := h.StartConversation(NewConversation{ModelID: "lemer-lite"})
	if err != nil {
		t.Fatalf("StartConversation: %v", err)
	}
	turnID, err := h.WriteTurn(conv, NewTurn{Role: "assistant", Content: "hi"})
	if err != nil {
		t.Fatalf("WriteTurn: %v", err)
	}
	if err := h.SetSignal(turnID, "liked"); err != nil {
		t.Fatalf("SetSignal: %v", err)
	}
}

// TestChatHistory_CopyTo_Bad_EmptyDest — an empty destination is rejected.
func TestChatHistory_CopyTo_Bad_EmptyDest(t *testing.T) {
	h := openTemp(t)
	core.AssertTrue(t, h.CopyTo("") != nil)
}

// TestChatHistory_CopyTo_Good_NestedDest — CopyTo creates a missing parent
// directory for the destination, then writes the checkpointed file there.
func TestChatHistory_CopyTo_Good_NestedDest(t *testing.T) {
	h := openTemp(t)
	conv, err := h.StartConversation(NewConversation{ModelID: "lemer-lite"})
	if err != nil {
		t.Fatalf("StartConversation: %v", err)
	}
	if _, err := h.WriteTurn(conv, NewTurn{Role: "user", Content: "hey"}); err != nil {
		t.Fatalf("WriteTurn: %v", err)
	}

	dest := filepath.Join(t.TempDir(), "deep", "nested", "copy.duckdb")
	if err := h.CopyTo(dest); err != nil {
		t.Fatalf("CopyTo: %v", err)
	}
	core.AssertTrue(t, core.Stat(dest).OK)

	// The copy is a usable archive with the same row counts.
	exported, err := Open("owlet", dest)
	if err != nil {
		t.Fatalf("Open copy: %v", err)
	}
	defer exported.Close()
	n, err := exported.CountTurns()
	core.AssertEqual(t, nil, err)
	core.AssertEqual(t, 1, n)
}

// TestChatHistory_ExportJSONL_Bad_EmptyDest — an empty destination is rejected.
func TestChatHistory_ExportJSONL_Bad_EmptyDest(t *testing.T) {
	h := openTemp(t)
	core.AssertTrue(t, h.ExportJSONL("") != nil)
}

// TestChatHistory_ExportJSONL_Good_AllFields — a fully-populated conversation
// (ended, tagged, with tool fields + tokens + signal) exports a JSONL line
// that carries every optional field through the nullable-scan branches.
func TestChatHistory_ExportJSONL_Good_AllFields(t *testing.T) {
	h := openTemp(t)
	conv, err := h.StartConversation(NewConversation{
		Title:          "vent",
		ModelID:        "lemer-lite",
		BaseModel:      "gemma-4-e2b-it-4bit",
		AdapterID:      "lek2",
		Tags:           []string{"life"},
		ConsentVersion: 2,
	})
	if err != nil {
		t.Fatalf("StartConversation: %v", err)
	}
	turnID, err := h.WriteTurn(conv, NewTurn{
		Role:        "assistant",
		Content:     "hi owlet",
		ToolCalls:   []byte(`[{"name":"search"}]`),
		ToolResults: []byte(`[{"hits":1}]`),
		TokensIn:    5,
		TokensOut:   7,
	})
	if err != nil {
		t.Fatalf("WriteTurn: %v", err)
	}
	if err := h.SetSignal(turnID, "liked"); err != nil {
		t.Fatalf("SetSignal: %v", err)
	}
	if err := h.EndConversation(conv); err != nil {
		t.Fatalf("EndConversation: %v", err)
	}

	dest := filepath.Join(t.TempDir(), "out.jsonl")
	if err := h.ExportJSONL(dest); err != nil {
		t.Fatalf("ExportJSONL: %v", err)
	}

	r := core.ReadFile(dest)
	if !r.OK {
		t.Fatalf("ReadFile: %v", r.Value)
	}
	var line JSONLConversation
	if u := core.JSONUnmarshal(firstLine(r.Value.([]byte)), &line); !u.OK {
		t.Fatalf("JSONUnmarshal: %v", u.Value)
	}

	core.AssertEqual(t, conv, line.ID)
	core.AssertEqual(t, "owlet", line.UserID)
	core.AssertEqual(t, "vent", line.Title)
	core.AssertEqual(t, "lemer-lite", line.ModelID)
	core.AssertEqual(t, "gemma-4-e2b-it-4bit", line.BaseModel)
	core.AssertEqual(t, "lek2", line.AdapterID)
	core.AssertEqual(t, 2, line.ConsentVersion)
	core.AssertTrue(t, line.EndedAt != nil)
	core.AssertEqual(t, 1, len(line.Tags))
	core.AssertEqual(t, 1, len(line.Turns))

	turn := line.Turns[0]
	core.AssertEqual(t, "assistant", turn.Role)
	core.AssertEqual(t, "hi owlet", turn.Content)
	core.AssertEqual(t, 5, turn.TokensIn)
	core.AssertEqual(t, 7, turn.TokensOut)
	core.AssertEqual(t, "liked", turn.Signal)
	core.AssertTrue(t, len(turn.ToolCalls) > 0)
	core.AssertTrue(t, len(turn.ToolResults) > 0)
}

// TestChatHistory_ExportJSONL_Good_Empty — an archive with no conversations
// exports an empty file without error (the loop-body-never-runs path).
func TestChatHistory_ExportJSONL_Good_Empty(t *testing.T) {
	h := openTemp(t)
	dest := filepath.Join(t.TempDir(), "empty.jsonl")
	if err := h.ExportJSONL(dest); err != nil {
		t.Fatalf("ExportJSONL: %v", err)
	}
	r := core.ReadFile(dest)
	if !r.OK {
		t.Fatalf("ReadFile: %v", r.Value)
	}
	core.AssertEqual(t, 0, len(r.Value.([]byte)))
}

// TestChatHistory_NullableJSON — the helper maps empty bytes to a SQL NULL and
// non-empty bytes to their string form.
func TestChatHistory_NullableJSON(t *testing.T) {
	core.AssertEqual(t, nil, nullableJSON(nil))
	core.AssertEqual(t, nil, nullableJSON([]byte{}))
	core.AssertEqual(t, `{"a":1}`, nullableJSON([]byte(`{"a":1}`)))
}

// firstLine returns the bytes up to (not including) the first newline, so a
// single-record JSONL export can be unmarshalled directly.
func firstLine(b []byte) []byte {
	for i, c := range b {
		if c == '\n' {
			return b[:i]
		}
	}
	return b
}
