# CoreAgent plugin — connecting to the MCP endpoint

The `core` plugin **connects to an already-running MCP endpoint over HTTP** — it
does **not** spawn a `core` binary. Reloading the plugin (or restarting Claude
Code) just reconnects; the agent never restarts.

## The endpoint

The plugin connects to **`http://127.0.0.1:9202/mcp`** (Streamable HTTP + SSE,
per-request Bearer auth). That is the **MCP plane of `lthn-agent hub`**:

- **Desktop (primary).** lthn/desktop's crew supervises `lthn-agent hub`
  automatically — the `CapabilitySandbox` member, control plane on `:9201`,
  MCP plane on `:9202`. Nothing to start: the endpoint is up while desktop runs.
- **Standalone (no desktop).**
  ```sh
  MCP_AUTH_TOKEN=<token> MCP_JWT_SECRET=<a-distinct-key> \
    lthn-agent hub --mcp-http 127.0.0.1:9202
  ```
  The MCP plane is **fail-closed**: it refuses to bind without `MCP_AUTH_TOKEN`
  **and** a distinct `MCP_JWT_SECRET`.

Either way the plugin hits the same `:9202/mcp` — "whichever is up."

## Auth

`.mcp.json` sends `Authorization: Bearer ${MCP_AUTH_TOKEN}`, so set
**`MCP_AUTH_TOKEN`** in the environment Claude Code sees — the same token the
endpoint runs with. The desktop crew resolves both secrets from `pkg/keys`
tier-0 before supervising the crew; standalone, export them yourself.

## Reload without restart

Because the plugin is a client, reloading it leaves `lthn-agent hub` — the
agent, its monitor, and any in-flight dispatch — running untouched. The old
stdio model coupled the agent's lifecycle to the plugin (reload = restart);
this severs that. (The old spawn-env, `MONITOR_INTERVAL` / `CORE_AGENT_DISPATCH`,
now belongs on the hub's launch, not the plugin config.)

## Install

Add the marketplace and install the `core` plugin, then make sure the endpoint
is up (desktop running, or the standalone hub above) with `MCP_AUTH_TOKEN` set
in the environment.
