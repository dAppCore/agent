# RFC-025: Agent Experience (AX) Design Principles

- **Status:** Draft
- **Authors:** Snider, Cladius
- **Date:** 2026-03-19
- **Applies to:** All Core ecosystem packages (CoreGO, CorePHP, CoreTS, core-agent)

## Abstract

Agent Experience (AX) is a design paradigm for software systems where the primary code consumer is an AI agent, not a human developer. AX sits alongside User Experience (UX) and Developer Experience (DX) as the third era of interface design.

This RFC establishes AX as a formal design principle for the Core ecosystem and defines the conventions that follow from it.

## Motivation

As of early 2026, AI agents write, review, and maintain the majority of code in the Core ecosystem. The original author has not manually edited code (outside of Core struct design) since October 2025. Code is processed semantically — agents reason about intent, not characters.

Design patterns inherited from the human-developer era optimise for the wrong consumer:

- **Short names** save keystrokes but increase semantic ambiguity
- **Functional option chains** are fluent for humans but opaque for agents tracing configuration
- **Error-at-every-call-site** produces 50% boilerplate that obscures intent
- **Generic type parameters** force agents to carry type context that the runtime already has
- **Panic-hiding conventions** (`Must*`) create implicit control flow that agents must special-case

AX acknowledges this shift and provides principles for designing code, APIs, file structures, and conventions that serve AI agents as first-class consumers.

## The Three Eras

| Era | Primary Consumer | Optimises For | Key Metric |
|-----|-----------------|---------------|------------|
| UX | End users | Discoverability, forgiveness, visual clarity | Task completion time |
| DX | Developers | Typing speed, IDE support, convention familiarity | Time to first commit |
| AX | AI agents | Predictability, composability, semantic navigation | Correct-on-first-pass rate |

AX does not replace UX or DX. End users still need good UX. Developers still need good DX. But when the primary code author and maintainer is an AI agent, the codebase should be designed for that consumer first.

## Principles

### 1. Predictable Names Over Short Names

Names are tokens that agents pattern-match across languages and contexts. Abbreviations introduce mapping overhead.

```
Config    not  Cfg
Service   not  Srv
Embed     not  Emb
Error     not  Err (as a subsystem name; err for local variables is fine)
Options   not  Opts
```

**Rule:** If a name would require a comment to explain, it is too short.

**Exception:** Industry-standard abbreviations that are universally understood (`HTTP`, `URL`, `ID`, `IPC`, `I18n`) are acceptable. The test: would an agent trained on any mainstream language recognise it without context?

### 2. Comments as Usage Examples

The function signature tells WHAT. The comment shows HOW with real values.

```go
// Detect the project type from files present
setup.Detect("/path/to/project")

// Set up a workspace with auto-detected template
setup.Run(setup.Options{Path: ".", Template: "auto"})

// Scaffold a PHP module workspace
setup.Run(setup.Options{Path: "./my-module", Template: "php"})
```

**Rule:** If a comment restates what the type signature already says, delete it. If a comment shows a concrete usage with realistic values, keep it.

**Rationale:** Agents learn from examples more effectively than from descriptions. A comment like "Run executes the setup process" adds zero information. A comment like `setup.Run(setup.Options{Path: ".", Template: "auto"})` teaches an agent exactly how to call the function.

### 3. Path Is Documentation

File and directory paths should be self-describing. An agent navigating the filesystem should understand what it is looking at without reading a README.

```
flow/deploy/to/homelab.yaml    — deploy TO the homelab
flow/deploy/from/github.yaml   — deploy FROM GitHub
flow/code/review.yaml           — code review flow
template/file/go/struct.go.tmpl — Go struct file template
template/dir/workspace/php/     — PHP workspace scaffold
```

**Rule:** If an agent needs to read a file to understand what a directory contains, the directory naming has failed.

**Corollary:** The unified path convention (folder structure = HTTP route = CLI command = test path) is AX-native. One path, every surface.

### 4. Templates Over Freeform

When an agent generates code from a template, the output is constrained to known-good shapes. When an agent writes freeform, the output varies.

```go
// Template-driven — consistent output
lib.RenderFile("php/action", data)
lib.ExtractDir("php", targetDir, data)

// Freeform — variance in output
"write a PHP action class that..."
```

**Rule:** For any code pattern that recurs, provide a template. Templates are guardrails for agents.

**Scope:** Templates apply to file generation, workspace scaffolding, config generation, and commit messages. They do NOT apply to novel logic — agents should write business logic freeform with the domain knowledge available.

### 5. Declarative Over Imperative

Agents reason better about declarations of intent than sequences of operations.

```yaml
# Declarative — agent sees what should happen
steps:
  - name: build
    flow: tools/docker-build
    with:
      context: "{{ .app_dir }}"
      image_name: "{{ .image_name }}"

  - name: deploy
    flow: deploy/with/docker
    with:
      host: "{{ .host }}"
```

```go
// Imperative — agent must trace execution
cmd := exec.Command("docker", "build", "--platform", "linux/amd64", "-t", imageName, ".")
cmd.Dir = appDir
if err := cmd.Run(); err != nil {
    return fmt.Errorf("docker build: %w", err)
}
```

**Rule:** Orchestration, configuration, and pipeline logic should be declarative (YAML/JSON). Implementation logic should be imperative (Go/PHP/TS). The boundary is: if an agent needs to compose or modify the logic, make it declarative.

### 6. Universal Types (Core Primitives)

Every component in the ecosystem accepts and returns the same primitive types. An agent processing any level of the tree sees identical shapes.

`Option` is a single key-value pair. `Options` is a collection. Any function that returns `Result` can accept `Options`.

```go
// Option — the atom
core.Option{K: "name", V: "brain"}

// Options — universal input (collection of Option)
core.Options{
    {K: "name", V: "myapp"},
    {K: "port", V: 8080},
}

// Result[T] — universal return
core.Result[*Embed]{Value: emb, OK: true}
```

Usage across subsystems — same shape everywhere:

```go
// Create Core
c := core.New(core.Options{{K: "name", V: "myapp"}})

// Mount embedded content
c.Data().New(core.Options{
    {K: "name", V: "brain"},
    {K: "source", V: brainFS},
    {K: "path", V: "prompts"},
})

// Register a transport handle
c.Drive().New(core.Options{
    {K: "name", V: "api"},
    {K: "transport", V: "https://api.lthn.ai"},
})

// Read back what was passed in
c.Options().String("name") // "myapp"
```

**Core primitive types:**

| Type | Purpose |
|------|---------|
| `core.Option` | Single key-value pair (the atom) |
| `core.Options` | Collection of Option (universal input) |
| `core.Result[T]` | Return value with OK/fail state (universal output) |
| `core.Config` | Runtime settings (what is active) |
| `core.Data` | Embedded or stored content from packages |
| `core.Drive` | Resource handle registry (transports) |
| `core.Service` | A managed component with lifecycle |

**Core struct subsystems:**

| Accessor | Analogy | Purpose |
|----------|---------|---------|
| `c.Options()` | argv | Input configuration used to create this Core |
| `c.Data()` | /mnt | Embedded assets mounted by packages |
| `c.Drive()` | /dev | Transport handles (API, MCP, SSH, VPN) |
| `c.Config()` | /etc | Configuration, settings, feature flags |
| `c.Fs()` | / | Local filesystem I/O (sandboxable) |
| `c.Error()` | — | Panic recovery and crash reporting (`ErrorPanic`) |
| `c.Log()` | — | Structured logging (`ErrorLog`) |
| `c.Service()` | — | Service registry and lifecycle |
| `c.Cli()` | — | CLI command framework |
| `c.IPC()` | — | Message bus |
| `c.I18n()` | — | Internationalisation |

**What this replaces:**

| Go Convention | Core AX | Why |
|--------------|---------|-----|
| `func With*(v) Option` | `core.Options{{K: k, V: v}}` | K/V pairs are parseable; option chains require tracing |
| `func Must*(v) T` | `core.Result[T]` | No hidden panics; errors flow through Core |
| `func *For[T](c) T` | `c.Service("name")` | String lookup is greppable; generics require type context |
| `val, err :=` everywhere | Single return via `core.Result` | Intent not obscured by error handling |
| `_ = err` | Never needed | Core handles all errors internally |
| `ErrPan` / `ErrLog` | `ErrorPanic` / `ErrorLog` | Full names — AX principle 1 |

## Applying AX to Existing Patterns

### File Structure

```
# AX-native: path describes content
core/agent/
├── go/                    # Go source
├── php/                   # PHP source
├── ui/                    # Frontend source
├── claude/                # Claude Code plugin
└── codex/                 # Codex plugin

# Not AX: generic names requiring README
src/
├── lib/
├── utils/
└── helpers/
```

### Error Handling

```go
// AX-native: errors are infrastructure, not application logic
svc := c.Service("brain")
cfg := c.Config().Get("database.host")
// Errors logged by Core. Code reads like a spec.

// Not AX: errors dominate the code
svc, err := c.ServiceFor[brain.Service]()
if err != nil {
    return fmt.Errorf("get brain service: %w", err)
}
cfg, err := c.Config().Get("database.host")
if err != nil {
    _ = err // silenced because "it'll be fine"
}
```

### API Design

```go
// AX-native: one shape, every surface
c := core.New(core.Options{
    {K: "name", V: "my-app"},
})
c.Service("process", processSvc)
c.Data().New(core.Options{{K: "name", V: "app"}, {K: "source", V: appFS}})

// Not AX: multiple patterns for the same thing
c, err := core.New(
    core.WithName("my-app"),
    core.WithService(factory1),
    core.WithAssets(appFS),
)
if err != nil { ... }
```

## Compatibility

AX conventions are valid, idiomatic Go/PHP/TS. They do not require language extensions, code generation, or non-standard tooling. An AX-designed codebase compiles, tests, and deploys with standard toolchains.

The conventions diverge from community patterns (functional options, Must/For, etc.) but do not violate language specifications. This is a style choice, not a fork.

## Adoption

AX applies to all new code in the Core ecosystem. Existing code migrates incrementally as it is touched — no big-bang rewrite.

Priority order:
1. **Public APIs** (package-level functions, struct constructors)
2. **File structure** (path naming, template locations)
3. **Internal fields** (struct field names, local variables)

## References

- dAppServer unified path convention (2024)
- CoreGO DTO pattern refactor (2026-03-18)
- Core primitives design (2026-03-19)
- Go Proverbs, Rob Pike (2015) — AX provides an updated lens

## Changelog

- 2026-03-20: Updated to match implementation — Option K/V atoms, Options as []Option, Data/Drive split, ErrorPanic/ErrorLog renames, subsystem table
- 2026-03-19: Initial draft
