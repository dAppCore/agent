package lifecycle

import (
	"database/sql"
	"encoding/json"
	"iter"
	"slices"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// SQLiteRegistry implements AgentRegistry using a SQLite database.
// It provides persistent storage that survives process restarts.
type SQLiteRegistry struct {
	db *sql.DB
	mu sync.Mutex // serialises read-modify-write operations
}

// NewSQLiteRegistry creates a new SQLite-backed agent registry at the given path.
// Use ":memory:" for tests that do not need persistence.
func NewSQLiteRegistry(dbPath string) (*SQLiteRegistry, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, &APIError{Code: 500, Message: "failed to open SQLite registry: " + err.Error()}
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, &APIError{Code: 500, Message: "failed to set WAL mode: " + err.Error()}
	}
	if _, err := db.Exec("PRAGMA busy_timeout=5000"); err != nil {
		db.Close()
		return nil, &APIError{Code: 500, Message: "failed to set busy_timeout: " + err.Error()}
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS agents (
		id             TEXT PRIMARY KEY,
		name           TEXT NOT NULL DEFAULT '',
		capabilities   TEXT NOT NULL DEFAULT '[]',
		status         TEXT NOT NULL DEFAULT 'available',
		last_heartbeat DATETIME NOT NULL DEFAULT (datetime('now')),
		current_load   INTEGER NOT NULL DEFAULT 0,
		max_load       INTEGER NOT NULL DEFAULT 0,
		registered_at  DATETIME NOT NULL DEFAULT (datetime('now'))
	)`); err != nil {
		db.Close()
		return nil, &APIError{Code: 500, Message: "failed to create agents table: " + err.Error()}
	}
	return &SQLiteRegistry{db: db}, nil
}

// Close releases the underlying SQLite database.
func (r *SQLiteRegistry) Close() error {
	return r.db.Close()
}

// Register adds or updates an agent in the registry. Returns an error if the
// agent ID is empty.
func (r *SQLiteRegistry) Register(agent AgentInfo) error {
	if agent.ID == "" {
		return &APIError{Code: 400, Message: "agent ID is required"}
	}
	caps, err := json.Marshal(agent.Capabilities)
	if err != nil {
		return &APIError{Code: 500, Message: "failed to marshal capabilities: " + err.Error()}
	}
	hb := agent.LastHeartbeat
	if hb.IsZero() {
		hb = time.Now().UTC()
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	_, err = r.db.Exec(`INSERT INTO agents (id, name, capabilities, status, last_heartbeat, current_load, max_load, registered_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			capabilities = excluded.capabilities,
			status = excluded.status,
			last_heartbeat = excluded.last_heartbeat,
			current_load = excluded.current_load,
			max_load = excluded.max_load`,
		agent.ID, agent.Name, string(caps), string(agent.Status), hb.Format(time.RFC3339Nano),
		agent.CurrentLoad, agent.MaxLoad, hb.Format(time.RFC3339Nano))
	if err != nil {
		return &APIError{Code: 500, Message: "failed to register agent: " + err.Error()}
	}
	return nil
}

// Deregister removes an agent from the registry. Returns an error if the agent
// is not found.
func (r *SQLiteRegistry) Deregister(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	res, err := r.db.Exec("DELETE FROM agents WHERE id = ?", id)
	if err != nil {
		return &APIError{Code: 500, Message: "failed to deregister agent: " + err.Error()}
	}
	n, err := res.RowsAffected()
	if err != nil {
		return &APIError{Code: 500, Message: "failed to check delete result: " + err.Error()}
	}
	if n == 0 {
		return &APIError{Code: 404, Message: "agent not found: " + id}
	}
	return nil
}

// Get returns a copy of the agent info for the given ID. Returns an error if
// the agent is not found.
func (r *SQLiteRegistry) Get(id string) (AgentInfo, error) {
	return r.scanAgent("SELECT id, name, capabilities, status, last_heartbeat, current_load, max_load FROM agents WHERE id = ?", id)
}

// List returns a copy of all registered agents.
func (r *SQLiteRegistry) List() []AgentInfo {
	return slices.Collect(r.All())
}

// All returns an iterator over all registered agents.
func (r *SQLiteRegistry) All() iter.Seq[AgentInfo] {
	return func(yield func(AgentInfo) bool) {
		rows, err := r.db.Query("SELECT id, name, capabilities, status, last_heartbeat, current_load, max_load FROM agents")
		if err != nil {
			return
		}
		defer rows.Close()

		for rows.Next() {
			a, err := r.scanAgentRow(rows)
			if err != nil {
				continue
			}
			if !yield(a) {
				return
			}
		}
	}
}

// Heartbeat updates the agent's LastHeartbeat timestamp. If the agent was
// Offline, it transitions to Available.
func (r *SQLiteRegistry) Heartbeat(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339Nano)

	// Update heartbeat for all agents, and transition offline agents to available.
	res, err := r.db.Exec(`UPDATE agents SET
		last_heartbeat = ?,
		status = CASE WHEN status = ? THEN ? ELSE status END
		WHERE id = ?`,
		now, string(AgentOffline), string(AgentAvailable), id)
	if err != nil {
		return &APIError{Code: 500, Message: "failed to heartbeat agent: " + err.Error()}
	}
	n, err := res.RowsAffected()
	if err != nil {
		return &APIError{Code: 500, Message: "failed to check heartbeat result: " + err.Error()}
	}
	if n == 0 {
		return &APIError{Code: 404, Message: "agent not found: " + id}
	}
	return nil
}

// Reap marks agents as Offline if their last heartbeat is older than ttl.
// Returns the IDs of agents that were reaped.
func (r *SQLiteRegistry) Reap(ttl time.Duration) []string {
	return slices.Collect(r.Reaped(ttl))
}

// Reaped returns an iterator over the IDs of agents that were reaped.
func (r *SQLiteRegistry) Reaped(ttl time.Duration) iter.Seq[string] {
	return func(yield func(string) bool) {
		r.mu.Lock()
		defer r.mu.Unlock()

		cutoff := time.Now().UTC().Add(-ttl).Format(time.RFC3339Nano)

		// Select agents that will be reaped before updating.
		rows, err := r.db.Query(
			"SELECT id FROM agents WHERE status != ? AND last_heartbeat < ?",
			string(AgentOffline), cutoff)
		if err != nil {
			return
		}
		defer rows.Close()

		var reaped []string
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				continue
			}
			reaped = append(reaped, id)
		}
		if err := rows.Err(); err != nil {
			return
		}
		rows.Close()

		if len(reaped) > 0 {
			// Build placeholders for IN clause.
			placeholders := make([]string, len(reaped))
			args := make([]any, len(reaped))
			for i, id := range reaped {
				placeholders[i] = "?"
				args[i] = id
			}
			query := "UPDATE agents SET status = ? WHERE id IN (" + strings.Join(placeholders, ",") + ")"
			allArgs := append([]any{string(AgentOffline)}, args...)
			_, _ = r.db.Exec(query, allArgs...)

			for _, id := range reaped {
				if !yield(id) {
					return
				}
			}
		}
	}
}

// --- internal helpers ---

// scanAgent executes a query that returns a single agent row.
func (r *SQLiteRegistry) scanAgent(query string, args ...any) (AgentInfo, error) {
	row := r.db.QueryRow(query, args...)
	var a AgentInfo
	var capsJSON string
	var statusStr string
	var hbStr string
	err := row.Scan(&a.ID, &a.Name, &capsJSON, &statusStr, &hbStr, &a.CurrentLoad, &a.MaxLoad)
	if err == sql.ErrNoRows {
		return AgentInfo{}, &APIError{Code: 404, Message: "agent not found: " + args[0].(string)}
	}
	if err != nil {
		return AgentInfo{}, &APIError{Code: 500, Message: "failed to scan agent: " + err.Error()}
	}
	if err := json.Unmarshal([]byte(capsJSON), &a.Capabilities); err != nil {
		return AgentInfo{}, &APIError{Code: 500, Message: "failed to unmarshal capabilities: " + err.Error()}
	}
	a.Status = AgentStatus(statusStr)
	a.LastHeartbeat, _ = time.Parse(time.RFC3339Nano, hbStr)
	return a, nil
}

// scanAgentRow scans a single row from a rows iterator.
func (r *SQLiteRegistry) scanAgentRow(rows *sql.Rows) (AgentInfo, error) {
	var a AgentInfo
	var capsJSON string
	var statusStr string
	var hbStr string
	err := rows.Scan(&a.ID, &a.Name, &capsJSON, &statusStr, &hbStr, &a.CurrentLoad, &a.MaxLoad)
	if err != nil {
		return AgentInfo{}, err
	}
	if err := json.Unmarshal([]byte(capsJSON), &a.Capabilities); err != nil {
		return AgentInfo{}, err
	}
	a.Status = AgentStatus(statusStr)
	a.LastHeartbeat, _ = time.Parse(time.RFC3339Nano, hbStr)
	return a, nil
}
