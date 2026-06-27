<!-- SPDX-License-Identifier: EUPL-1.2 -->
# Extending — Go & PHP

## Adding Go functionality

### New package

Create a directory under `go/pkg/`. Follow the convention — one test file per source file,
`*_example_test.go` doubling as runnable examples. Import as
`dappco.re/go/agent/pkg/<name>`.

### New CLI command

Commands register against `core.Core` via `c.Command(name, core.Command{...})`. Binary
commands go in `go/cmd/core-agent/commands.go`; subsystem commands in the owning package
(e.g. `pkg/agentic/commands_plan.go`):

```go
c.Command("my-command", core.Command{
    Description: "What it does",
    Action: func(opts core.Options) core.Result {
        return core.Result{OK: true}
    },
})
```

### New MCP tool

Tools register into the shared `dappco.re/go/mcp` service via `coremcp.AddToolRecorded`:

```go
coremcp.AddToolRecorded(svc, svc.Server(), "<subsystem>", &mcp.Tool{
    Name:        "my_tool",
    Description: "What the tool does and when to use it.",
}, func(ctx context.Context, req *mcp.CallToolRequest, in MyInput) (*mcp.CallToolResult, MyOutput, error) {
    return nil, MyOutput{...}, nil
})
```

Wire it from the subsystem's `RegisterTools` (see `pkg/agentic/dispatch.go` or
`cmd/core-agent/lemma_mcp.go`). The same service serves both `mcp` (stdio) and `serve`
(HTTP).

## Adding PHP functionality

All PHP uses the `Core\Mod\Agentic\*` namespace.

- **Model** → `src/php/Models/` (`Core\Mod\Agentic\Models`), extends Eloquent `Model`.
- **Action** → `src/php/Actions/`, single-purpose with the `Action` concern
  (`DoSomething::run('hello')`).
- **Controller** → `src/php/Controllers/`; routes in `src/php/Routes/api.php` (loaded by
  `onApiRoutes`).
- **Artisan command** → `src/php/Console/Commands/`, registered in `Boot::onConsole()`.
- **Livewire component** → `src/php/View/Modal/Admin/` (+ Blade in `View/Blade/admin/`),
  registered in `Boot::onAdminPanel()` via `$event->livewire(...)`.

See [plugins](plugins.md) for extending the provider/plugin side.
