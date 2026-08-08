// SPDX-License-Identifier: EUPL-1.2

package chathistory

import (
	"database/sql"
	"encoding/json"
	"io"
	"time"

	core "dappco.re/go"
)

// CopyTo copies the live DuckDB file to dest. The user-friendly export
// path: hand them a single .duckdb they can open in any tool. The
// source file is checkpointed first to ensure all WAL writes are
// flushed into the main file.
//
// This is the simplest export — the file IS the format. For tools
// that prefer line-delimited records, ExportJSONL.
//
//	if err := h.CopyTo("/Users/owlet/Downloads/owlet-chats-2026-05-26.duckdb"); err != nil { ... }
func (h *History) CopyTo(dest string) error {
	if h == nil || h.db == nil {
		return core.E("chathistory.CopyTo", "history closed", nil)
	}
	if core.Trim(dest) == "" {
		return core.E("chathistory.CopyTo", "dest required", nil)
	}
	if _, err := h.db.Exec(`CHECKPOINT`); err != nil {
		return core.E("chathistory.CopyTo", "checkpoint", err)
	}
	srcResult := core.Open(h.path)
	if !srcResult.OK {
		return core.E("chathistory.CopyTo", "open source", srcResult.Value.(error))
	}
	src := srcResult.Value.(*core.OSFile)
	defer func() { _ = src.Close() }() // read handle: nothing to lose on close
	if dir := core.PathDir(dest); dir != "" {
		if r := core.MkdirAll(dir, 0o755); !r.OK {
			return core.E("chathistory.CopyTo", "mkdir dest parent", r.Value.(error))
		}
	}
	dstResult := core.Create(dest)
	if !dstResult.OK {
		return core.E("chathistory.CopyTo", "create dest", dstResult.Value.(error))
	}
	dst := dstResult.Value.(*core.OSFile)
	// Net for the error paths below; the success path closes explicitly.
	defer func() { _ = dst.Close() }()
	if _, err := io.Copy(dst, src); err != nil {
		return core.E("chathistory.CopyTo", "copy bytes", err)
	}
	// Checked, not deferred-and-forgotten: a write handle that fails to close
	// may never have flushed, and returning nil there reports a copy that did
	// not fully happen.
	if err := dst.Close(); err != nil {
		return core.E("chathistory.CopyTo", "close dest", err)
	}
	return nil
}

// JSONLConversation is one record line in the JSONL export. Shape is
// self-describing — any tool that reads JSONL can consume the archive
// without DuckDB. Future LoRA training data prep should prefer the
// .duckdb (richer query surface), but JSONL is the non-technical
// user's option.
type JSONLConversation struct {
	ID             string      `json:"id"`
	UserID         string      `json:"user_id"`
	Title          string      `json:"title,omitempty"`
	StartedAt      time.Time   `json:"started_at"`
	EndedAt        *time.Time  `json:"ended_at,omitempty"`
	ModelID        string      `json:"model_id,omitempty"`
	BaseModel      string      `json:"base_model,omitempty"`
	AdapterID      string      `json:"adapter_id,omitempty"`
	Tags           []string    `json:"tags,omitempty"`
	ConsentVersion int         `json:"consent_version"`
	Turns          []JSONLTurn `json:"turns"`
}

// JSONLTurn is one message inside a conversation's `turns` array.
type JSONLTurn struct {
	ID          string          `json:"id"`
	Ordinal     int             `json:"ordinal"`
	Role        string          `json:"role"`
	Content     string          `json:"content"`
	ToolCalls   json.RawMessage `json:"tool_calls,omitempty"`
	ToolResults json.RawMessage `json:"tool_results,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	TokensIn    int             `json:"tokens_in,omitempty"`
	TokensOut   int             `json:"tokens_out,omitempty"`
	Signal      string          `json:"signal,omitempty"`
}

// ExportJSONL writes one conversation per line to dest. Each line is
// a JSONLConversation with all turns inlined. Order is by started_at.
//
//	if err := h.ExportJSONL("/Users/owlet/Downloads/owlet-chats.jsonl"); err != nil { ... }
func (h *History) ExportJSONL(dest string) error {
	if h == nil || h.db == nil {
		return core.E("chathistory.ExportJSONL", "history closed", nil)
	}
	if core.Trim(dest) == "" {
		return core.E("chathistory.ExportJSONL", "dest required", nil)
	}
	if dir := core.PathDir(dest); dir != "" {
		if r := core.MkdirAll(dir, 0o755); !r.OK {
			return core.E("chathistory.ExportJSONL", "mkdir dest parent", r.Value.(error))
		}
	}
	fResult := core.Create(dest)
	if !fResult.OK {
		return core.E("chathistory.ExportJSONL", "create dest", fResult.Value.(error))
	}
	f := fResult.Value.(*core.OSFile)
	// Net for the error paths below; the success path closes explicitly.
	defer func() { _ = f.Close() }()

	convRows, err := h.db.Query(
		`SELECT id, user_id, title, started_at, ended_at, model_id, base_model,
		        adapter_id, tags, consent_version
		   FROM conversations
		  ORDER BY started_at`,
	)
	if err != nil {
		return core.E("chathistory.ExportJSONL", "query conversations", err)
	}
	defer func() { _ = convRows.Close() }() // read cursor

	for convRows.Next() {
		var c JSONLConversation
		var title, modelID, baseModel, adapterID sql.NullString
		var endedAt sql.NullTime
		var tagsJSON sql.NullString
		if err := convRows.Scan(
			&c.ID, &c.UserID, &title, &c.StartedAt, &endedAt,
			&modelID, &baseModel, &adapterID, &tagsJSON, &c.ConsentVersion,
		); err != nil {
			return core.E("chathistory.ExportJSONL", "scan conversation", err)
		}
		c.Title = title.String
		c.ModelID = modelID.String
		c.BaseModel = baseModel.String
		c.AdapterID = adapterID.String
		if endedAt.Valid {
			c.EndedAt = &endedAt.Time
		}
		if tagsJSON.Valid && tagsJSON.String != "" {
			_ = core.JSONUnmarshal([]byte(tagsJSON.String), &c.Tags)
		}

		turnRows, err := h.db.Query(
			`SELECT id, ordinal, role, content, tool_calls, tool_results,
			        created_at, tokens_in, tokens_out, signal
			   FROM turns
			  WHERE conversation_id = ?
			  ORDER BY ordinal`,
			c.ID,
		)
		if err != nil {
			return core.E("chathistory.ExportJSONL", "query turns", err)
		}
		for turnRows.Next() {
			var t JSONLTurn
			var toolCalls, toolResults sql.NullString
			var tokensIn, tokensOut sql.NullInt32
			var signal sql.NullString
			if err := turnRows.Scan(
				&t.ID, &t.Ordinal, &t.Role, &t.Content,
				&toolCalls, &toolResults, &t.CreatedAt,
				&tokensIn, &tokensOut, &signal,
			); err != nil {
				_ = turnRows.Close() // read cursor; the scan error is what matters
				return core.E("chathistory.ExportJSONL", "scan turn", err)
			}
			if toolCalls.Valid {
				t.ToolCalls = json.RawMessage(toolCalls.String)
			}
			if toolResults.Valid {
				t.ToolResults = json.RawMessage(toolResults.String)
			}
			if tokensIn.Valid {
				t.TokensIn = int(tokensIn.Int32)
			}
			if tokensOut.Valid {
				t.TokensOut = int(tokensOut.Int32)
			}
			t.Signal = signal.String
			c.Turns = append(c.Turns, t)
		}
		_ = turnRows.Close() // read cursor

		marshalled := core.JSONMarshal(c)
		if !marshalled.OK {
			return core.E("chathistory.ExportJSONL", "marshal conversation", marshalled.Value.(error))
		}
		line := marshalled.Value.([]byte)
		if _, err := f.Write(line); err != nil {
			return core.E("chathistory.ExportJSONL", "write line", err)
		}
		if _, err := f.Write([]byte{'\n'}); err != nil {
			return core.E("chathistory.ExportJSONL", "write newline", err)
		}
	}

	// Checked, not deferred-and-forgotten: every line above can be written
	// successfully and still be lost if the final flush fails, which would
	// report a complete export of a truncated file.
	if err := f.Close(); err != nil {
		return core.E("chathistory.ExportJSONL", "close dest", err)
	}

	return nil
}
