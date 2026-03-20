# TODO.md - core-agentic

Production-quality task list for the AI agent orchestration package.

**Last updated:** 2026-01-29

---

## P1 - Critical / Security

### Security Hardening

- [x] **SEC-001: API key hashing uses SHA-256 without salt** (FIXED 2026-01-29)
  - Location: `Models/AgentApiKey.php::generate()`
  - Risk: Weak credential storage vulnerable to rainbow table attacks
  - Fix: Switched to `password_hash()` with Argon2id
  - Note: `findByKey()` now iterates active keys since Argon2id uses unique salts
  - Added `verifyKey()` method for single-key verification

- [x] **SEC-002: SQL injection risk in TaskCommand orderByRaw** (FIXED 2026-01-29)
  - Location: `Console/Commands/TaskCommand.php`
  - Risk: `orderByRaw("FIELD(priority, ...)")` pattern vulnerable if extended
  - Fix: Replaced with parameterised `orderByPriority()` and `orderByStatus()` scopes
  - Also fixed `PlanCommand.php` with same pattern

- [x] **SEC-003: StateSet tool lacks workspace scoping** (FIXED 2026-01-29)
  - Location: `Mcp/Tools/Agent/State/StateSet.php`
  - Risk: Plan lookup by slug without workspace constraint - cross-tenant data access
  - Fix: Added workspace_id context check and forWorkspace() scoping to:
    - `StateSet.php`, `StateGet.php`, `StateList.php`
    - `PlanGet.php`, `PlanList.php`
  - Added ToolDependency for workspace_id requirement

- [ ] **SEC-004: Missing rate limiting on MCP tool execution**
  - Location: `Services/AgentToolRegistry.php::execute()`
  - Risk: API key rate limits apply to auth, not individual tool calls
  - Fix: Add per-tool rate limiting in execute() method
  - Acceptance: Tool execution respects rate limits per workspace

### Input Validation

- [ ] **VAL-001: Template variable injection vulnerability**
  - Location: `Services/PlanTemplateService.php::substituteVariables()`
  - Risk: Special characters in variables could corrupt JSON structure
  - Status: Partially fixed with escapeForJson, but needs additional input sanitisation
  - Fix: Validate variable values against allowed character sets
  - Acceptance: Malicious variable values rejected with clear error

---

## P2 - High Priority

### Test Coverage (Critical Gap)

- [x] **TEST-001: Add AgentApiKey model tests** (FIXED 2026-01-29)
  - Created `tests/Feature/AgentApiKeyTest.php` using Pest functional syntax
  - Created `tests/Pest.php` for Pest configuration with helper functions
  - Covers: key generation with Argon2id, validation, permissions, rate limiting, IP restrictions
  - Additional coverage: key rotation, security edge cases
  - 70+ test cases for comprehensive model coverage

- [x] **TEST-002: Add AgentApiKeyService tests** (FIXED 2026-01-29)
  - Created `tests/Feature/AgentApiKeyServiceTest.php` using Pest functional syntax
  - Covers: authenticate(), IP validation, rate limit tracking, key management
  - 58 test cases including full authentication flow and edge cases

- [x] **TEST-003: Add IpRestrictionService tests** (FIXED 2026-01-29)
  - Created `tests/Feature/IpRestrictionServiceTest.php` using Pest functional syntax
  - Covers: IPv4/IPv6 validation, CIDR matching (all prefix lengths), edge cases
  - 78 test cases for security-critical IP whitelisting

- [x] **TEST-004: Add PlanTemplateService tests** (FIXED 2026-01-29)
  - Created `tests/Feature/PlanTemplateServiceTest.php` using Pest functional syntax
  - Covers: template listing, retrieval, preview, variable substitution, plan creation, validation, categories, context generation
  - 47 test cases organised into 9 describe blocks with proper beforeEach/afterEach setup

- [x] **TEST-005: Add AI provider service tests** (FIXED 2026-01-29)
  - Created `tests/Unit/ClaudeServiceTest.php` - Anthropic Claude API tests (Pest functional syntax)
  - Created `tests/Unit/GeminiServiceTest.php` - Google Gemini API tests (Pest functional syntax)
  - Created `tests/Unit/OpenAIServiceTest.php` - OpenAI API tests (Pest functional syntax)
  - Created `tests/Unit/AgenticManagerTest.php` - Provider coordinator tests (Pest functional syntax)
  - All use mocked HTTP responses with describe/it blocks
  - Covers: provider configuration, API key management, request handling, response parsing
  - Includes: error handling, retry logic (429/500), edge cases, streaming
  - AgenticManager tests: provider registration, retrieval, availability, default provider handling

### Missing Database Infrastructure

- [x] **DB-001: Missing agent_plans table migration** (FIXED 2026-01-29)
  - Created `Migrations/0001_01_01_000003_create_agent_plans_tables.php`
  - Creates: `agent_plans`, `agent_phases`, `agent_workspace_states` tables
  - Adds `agent_plan_id` FK and related columns to `agent_sessions`
  - Includes proper indexes for slug, workspace, and status queries

- [x] **DB-002: Missing indexes on frequently queried columns** (FIXED 2026-02-23)
  - `agent_sessions.session_id` - unique() constraint creates implicit index; sufficient for lookups
  - `agent_plans.slug` - redundant plain index dropped; compound (workspace_id, slug) index added
  - `workspace_states.key` - already indexed via ->index('key') in migration 000003

### Error Handling

- [ ] **ERR-001: ClaudeService stream() lacks error handling**
  - Location: `Services/ClaudeService.php::stream()`
  - Issue: No try/catch around streaming, could fail silently
  - Fix: Wrap in exception handling, yield error events

- [x] **ERR-002: ContentService has no batch failure recovery** (FIXED 2026-02-23)
  - Location: `Services/ContentService.php::generateBatch()`
  - Issue: Failed articles stop processing, no resume capability
  - Fix: Added progress state file, per-article retry (maxRetries param), `resumeBatch()` method
  - Tests: 6 new tests in `tests/Feature/ContentServiceTest.php` covering state persistence, resume, retries

---

## P3 - Medium Priority

### Developer Experience

- [x] **DX-001: Missing workspace context error messages unclear** (FIXED 2026-02-23)
  - Location: Multiple MCP tools
  - Issue: "workspace_id is required" didn't explain how to fix
  - Fix: Updated error messages in PlanCreate, PlanGet, PlanList, StateSet, StateGet, StateList, SessionStart to include actionable guidance and link to documentation

- [x] **DX-002: AgenticManager doesn't validate API keys on init** (FIXED 2026-02-23)
  - Location: `Services/AgenticManager.php::registerProviders()`
  - Issue: Empty API key creates provider that fails on first use
  - Fix: `Log::warning()` emitted for each provider registered without an API key; message names the env var to set

- [x] **DX-003: Plan template variable errors not actionable** (FIXED 2026-02-23)
  - Location: `Services/PlanTemplateService.php::validateVariables()`
  - Fix: Error messages now include variable description, example/examples, and expected format
  - Added `naming_convention` field to result; extracted `buildVariableError()` helper
  - New tests: description in error, example value, multiple examples, format hint, naming_convention field

### Code Quality

- [x] **CQ-001: Duplicate state models (WorkspaceState vs AgentWorkspaceState)** (FIXED 2026-02-23)
  - Deleted `Models/AgentWorkspaceState.php` (unused legacy port)
  - Consolidated into `Models/WorkspaceState.php` backed by `agent_workspace_states` table
  - Updated `AgentPlan`, `StateSet`, `SecurityTest` to use `WorkspaceState`
  - Added `WorkspaceStateTest` covering model behaviour and static helpers

- [x] **CQ-002: ApiKeyManager uses Core\Api\ApiKey, not AgentApiKey** (FIXED 2026-02-23)
  - Location: `View/Modal/Admin/ApiKeyManager.php`
  - Issue: Livewire component uses different API key model than services
  - Fix: Switched to `AgentApiKey` model and `AgentApiKeyService` throughout
  - Updated blade template to use `permissions`, `getMaskedKey()`, `togglePermission()`
  - Added integration tests in `tests/Feature/ApiKeyManagerTest.php`

- [x] **CQ-003: ForAgentsController cache key not namespaced** (FIXED 2026-02-23)
  - Location: `Controllers/ForAgentsController.php`
  - Issue: `Cache::remember('agentic.for-agents.json', ...)` could collide
  - Fix: Cache key and TTL now driven by `mcp.cache.for_agents_key` / `mcp.cache.for_agents_ttl` config
  - Added `cacheKey()` public method and config entries in `config.php`
  - Tests added in `tests/Feature/ForAgentsControllerTest.php`

### Performance

- [ ] **PERF-001: AgentPhase::checkDependencies does N queries**
  - Location: `Models/AgentPhase.php::checkDependencies()`
  - Issue: Loops through dependencies with individual `find()` calls
  - Fix: Eager load or use whereIn for batch lookup

- [ ] **PERF-002: AgentToolRegistry::forApiKey iterates all tools**
  - Location: `Services/AgentToolRegistry.php::forApiKey()`
  - Issue: O(n) filter on every request
  - Fix: Cache permitted tools per API key

---

## P4 - Low Priority

### Documentation Gaps

- [x] **DOC-001: Add PHPDoc to AgentDetection patterns** (FIXED 2026-02-23)
  - Location: `Services/AgentDetection.php`
  - Issue: User-Agent patterns undocumented
  - Fix: Added PHPDoc with real UA examples to all pattern constants, class-level usage examples, and MCP_TOKEN_HEADER docs

- [ ] **DOC-002: Document MCP tool dependency system**
  - Location: `Mcp/Tools/Agent/` directory
  - Fix: Add README explaining ToolDependency, context requirements

### Feature Completion

- [ ] **FEAT-001: Session cleanup for stale sessions**
  - Issue: No mechanism to clean up abandoned sessions
  - Fix: Add scheduled command to mark stale sessions as failed
  - Criteria: Sessions inactive >24h marked as abandoned

- [ ] **FEAT-002: Plan archival with data retention policy**
  - Issue: Archived plans kept forever
  - Fix: Add configurable retention period, cleanup job

- [x] **FEAT-003: Template version management**
  - Location: `Services/PlanTemplateService.php`, `Models/PlanTemplateVersion.php`
  - Issue: Template changes affect existing plan references
  - Fix: Add version tracking to templates — implemented in #35

### Consistency

- [x] **CON-001: Mixed UK/US spelling in code comments** (FIXED 2026-02-23)
  - Issue: Some comments use "organize" instead of "organise"
  - Fix: Audit and fix to UK English per CLAUDE.md
  - Changed: `Mcp/Servers/Marketing.php` "Organize" → "Organise" in docstring

- [ ] **CON-002: Inconsistent error response format**
  - Issue: Some tools return `['error' => ...]`, others `['success' => false, ...]`
  - Fix: Standardise on single error response format

---

## P5 - Nice to Have

### Observability

- [ ] **OBS-001: Add structured logging to AI provider calls**
  - Issue: No visibility into API call timing, token usage
  - Fix: Add Log::info with provider, model, tokens, latency

- [ ] **OBS-002: Add Prometheus metrics for tool execution**
  - Fix: Emit tool_execution_seconds, tool_errors_total

### Admin UI Improvements

- [ ] **UI-001: Add bulk operations to plan list**
  - Fix: Multi-select archive, activate actions

- [ ] **UI-002: Add session timeline visualisation**
  - Fix: Visual work_log display with timestamps

- [ ] **UI-003: Add template preview before creation**
  - Fix: Show resolved variables, phase list

---

## P6 - Future / Backlog

### Architecture Evolution

- [ ] **ARCH-001: Consider event sourcing for session work_log**
  - Benefit: Full replay capability, audit trail
  - Consideration: Adds complexity

- [ ] **ARCH-002: Extract AI provider abstraction to separate package**
  - Benefit: Reusable across other modules
  - Consideration: Increases package count

### Integration

- [ ] **INT-001: Add webhook notifications for plan status changes**
  - Use: External integrations can react to agent progress

- [ ] **INT-002: Add Slack/Discord integration for session alerts**
  - Use: Team visibility into agent operations

---

## Completed Items

### Security (Fixed)

- [x] Missing `agent_api_keys` table migration - Migration added
- [x] Rate limiting bypass - getRecentCallCount now reads from cache
- [x] Admin routes lack middleware - RequireHades applied
- [x] ForAgentsController missing rate limiting - Added
- [x] SEC-001: API key hashing SHA-256 to Argon2id - Switched to password_hash() (2026-01-29)
- [x] SEC-002: SQL injection in orderByRaw - Replaced with parameterised scopes (2026-01-29)
- [x] SEC-003: StateSet/StateGet/StateList/PlanGet/PlanList workspace scoping - Added forWorkspace() checks (2026-01-29)

### Code Quality (Fixed)

- [x] Add retry logic to AI provider services - HasRetry trait added
- [x] Stream parsing fragile - HasStreamParsing trait added
- [x] ContentService hardcoded paths - Now configurable
- [x] Rate limit TTL race condition - Uses Cache::add()
- [x] JSON escaping in template substitution - Added

### DX (Fixed)

- [x] MCP tool handlers commented out - Documented properly
- [x] MCP token lookup not implemented - Database lookup added

### Test Coverage (Fixed)

- [x] TEST-001: AgentApiKey model tests - 70+ tests in AgentApiKeyTest.php (2026-01-29)
- [x] TEST-002: AgentApiKeyService tests - 58 tests in AgentApiKeyServiceTest.php (2026-01-29)
- [x] TEST-003: IpRestrictionService tests - 78 tests in IpRestrictionServiceTest.php (2026-01-29)
- [x] TEST-004: PlanTemplateService tests - 35+ tests in PlanTemplateServiceTest.php (2026-01-29)
- [x] TEST-005: AI provider tests - ClaudeServiceTest, GeminiServiceTest, OpenAIServiceTest, AgenticManagerTest (2026-01-29)

### Database (Fixed)

- [x] DB-001: Missing agent_plans migration - Created 0001_01_01_000003_create_agent_plans_tables.php (2026-01-29)
- [x] DB-002: Performance indexes - Dropped redundant slug index, added compound (workspace_id, slug) index (2026-02-23)

---

## Notes

**Test Coverage Estimate:** ~65% (improved from ~35%)
- Models: Well tested (AgentPlan, AgentPhase, AgentSession, AgentApiKey)
- Services: AgentApiKeyService, IpRestrictionService, PlanTemplateService now tested
- AI Providers: ClaudeService, GeminiService, OpenAIService, AgenticManager unit tested
- Commands: Untested (3 commands)
- Livewire: Untested

**Priority Guide:**
- P1: Security/data integrity - fix before production
- P2: High impact on reliability - fix in next sprint
- P3: Developer friction - address during regular work
- P4: Nice to have - backlog candidates
- P5: Polish - when time permits
- P6: Future considerations - parking lot
