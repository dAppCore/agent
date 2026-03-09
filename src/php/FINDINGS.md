# Phase 0: Environment Assessment + Test Baseline

**Date:** 2026-02-20
**Branch:** feat/phase-0-assessment
**Issue:** #1
**Agent:** Clotho <clotho@lthn.ai>

---

## Executive Summary

This phase 0 assessment provides a comprehensive baseline of the `host-uk/core-agentic` Laravel package. The package implements AI agent orchestration with MCP (Model Context Protocol) tools, multi-agent collaboration, and unified AI provider access.

**Key Findings:**
- ✅ Well-structured event-driven architecture
- ✅ Comprehensive test coverage (~65%) with Pest framework
- ✅ Security-conscious design with recent hardening (Jan 2026)
- ⚠️ Cannot run tests without `host-uk/core` dependency
- ⚠️ Some code quality issues identified in existing TODO.md
- ✅ Excellent documentation and conventions

---

## 1. Environment Assessment

### 1.1 Dependency Constraints

**Status:** ⚠️ BLOCKED - Cannot install dependencies

```bash
$ composer install --no-interaction
# Error: host-uk/core dev-main could not be found
```

**Root Cause:**
- Package depends on `host-uk/core` (dev-main) which is a private dependency
- No composer.lock file present
- Missing repository configuration for private packages

**Impact:**
- Cannot run test suite (`composer test` / `vendor/bin/pest`)
- Cannot run linter (`composer run lint` / `vendor/bin/pint`)
- Cannot run static analysis (`vendor/bin/phpstan`)

**Recommendation:**
- Add repository configuration to composer.json for host-uk/core
- OR provide mock/stub for testing in isolation
- OR test within a full host application environment

### 1.2 Codebase Metrics

| Metric | Count |
|--------|-------|
| Total PHP files | 125 |
| Models | 9 |
| Services | 15 |
| MCP Tools | 34 |
| Migrations | 3 |
| Tests | 15+ test files |
| Console Commands | 3 |
| Livewire Components | 9 |

### 1.3 Test Infrastructure

**Framework:** Pest 3.x (functional syntax)

**Configuration:**
- `tests/Pest.php` - Central configuration
- `tests/TestCase.php` - Orchestra Testbench base class
- RefreshDatabase trait applied to Feature tests

**Test Coverage Breakdown (from TODO.md):**
```
Current: ~65% (improved from ~35% in Jan 2026)

✅ Models: Well tested
  - AgentPlan, AgentPhase, AgentSession, AgentApiKey

✅ Services: Partially tested
  - AgentApiKeyService (58 tests)
  - IpRestrictionService (78 tests)
  - PlanTemplateService (47 tests)
  - AI Providers: ClaudeService, GeminiService, OpenAIService, AgenticManager

❌ Untested:
  - 3 Console Commands
  - 9 Livewire Components
  - Some MCP Tools
```

**Test Files:**
```
tests/
├── Feature/
│   ├── AgentApiKeyTest.php (70+ tests)
│   ├── AgentApiKeyServiceTest.php (58 tests)
│   ├── IpRestrictionServiceTest.php (78 tests)
│   ├── PlanTemplateServiceTest.php (47 tests)
│   ├── ContentServiceTest.php
│   ├── AgentPlanTest.php
│   ├── AgentPhaseTest.php
│   ├── AgentSessionTest.php
│   └── SecurityTest.php
├── Unit/
│   ├── ClaudeServiceTest.php
│   ├── GeminiServiceTest.php
│   ├── OpenAIServiceTest.php
│   └── AgenticManagerTest.php
└── UseCase/
    └── AdminPanelBasic.php
```

---

## 2. Architecture Review

### 2.1 Boot System (Event-Driven)

**Pattern:** Event-driven lazy loading via Laravel service provider

```php
// Boot.php - Responds to Core framework events
public static array $listens = [
    AdminPanelBooting::class => 'onAdminPanel',
    ConsoleBooting::class => 'onConsole',
    McpToolsRegistering::class => 'onMcpTools',
];
```

**Strengths:**
- ✅ Lazy loading reduces unnecessary overhead
- ✅ Clean separation of concerns
- ✅ Follows Laravel conventions
- ✅ Event handlers are well-documented

**Rate Limiting:**
- Configured in Boot::configureRateLimiting()
- 60 requests/minute per IP for agentic-api
- Separate API key-based rate limiting in AgentApiKey model

### 2.2 MCP Tools Architecture

**Location:** `Mcp/Tools/Agent/` organised by domain

**Structure:**
```
Mcp/Tools/Agent/
├── AgentTool.php (Base class)
├── Contracts/
│   └── AgentToolInterface.php
├── Plan/ (5 tools)
├── Phase/ (3 tools)
├── Session/ (9 tools)
├── State/ (3 tools)
├── Task/ (2 tools)
├── Content/ (8 tools)
└── Template/ (3 tools)
```

**Base Class Features (AgentTool):**
- Input validation helpers (requireString, optionalInt, requireEnum)
- Circuit breaker protection via withCircuitBreaker()
- Standardised response format (success(), error())

**Design Pattern:**
```php
abstract class AgentTool implements AgentToolInterface
{
    protected function requireString(array $input, string $key): string;
    protected function optionalInt(array $input, string $key, ?int $default = null): ?int;
    protected function requireEnum(array $input, string $key, array $allowed): string;
    protected function withCircuitBreaker(callable $callback);
    protected function success(array $data): array;
    protected function error(string $message, int $code = 400): array;
}
```

**Strengths:**
- ✅ Clean abstraction with consistent interface
- ✅ Built-in validation helpers
- ✅ Circuit breaker for resilience
- ✅ Domain-driven organisation

**Potential Issues:**
- ⚠️ No per-tool rate limiting (noted in TODO.md SEC-004)
- ⚠️ Workspace scoping added recently (SEC-003 - Jan 2026)

### 2.3 AI Provider System

**Manager:** `AgenticManager` (singleton)

**Supported Providers:**
1. Claude (Anthropic) - `ClaudeService`
2. Gemini (Google) - `GeminiService`
3. OpenAI - `OpenAIService`

**Usage Pattern:**
```php
$ai = app(AgenticManager::class);
$ai->claude()->generate($prompt);
$ai->gemini()->generate($prompt);
$ai->openai()->generate($prompt);
$ai->provider('gemini')->generate($prompt);
```

**Shared Concerns (Traits):**
- `HasRetry` - Automatic retry with exponential backoff
- `HasStreamParsing` - Server-sent events (SSE) parsing

**Strengths:**
- ✅ Clean provider abstraction
- ✅ Consistent interface across providers
- ✅ Built-in retry logic
- ✅ Streaming support

**Identified Issues (from TODO.md):**
- ⚠️ DX-002: No API key validation on init (provider fails on first use)
- ⚠️ ERR-001: ClaudeService stream() lacks error handling

### 2.4 Data Model Design

**Core Models:**

| Model | Purpose | Relationships |
|-------|---------|---------------|
| `AgentPlan` | Work plans with phases | hasMany AgentPhase, hasMany AgentSession |
| `AgentPhase` | Plan phases with tasks | belongsTo AgentPlan, hasMany Task |
| `AgentSession` | Agent execution sessions | belongsTo AgentPlan, has work_log JSON |
| `AgentApiKey` | API key management | belongsTo Workspace, has permissions array |
| `WorkspaceState` | Key-value state storage | belongsTo Workspace |
| `AgentWorkspaceState` | (Duplicate?) | - |

**Schema Features:**
- Status enums: `pending`, `in_progress`, `completed`, `failed`, `abandoned`
- JSON columns: work_log, context, permissions, metadata
- Soft deletes on plans and sessions
- Timestamps on all models

**Identified Issues:**
- ⚠️ CQ-001: Duplicate state models (WorkspaceState vs AgentWorkspaceState)
- ⚠️ DB-002: Missing indexes on frequently queried columns (slug, session_id, key)
- ⚠️ PERF-001: AgentPhase::checkDependencies does N queries

### 2.5 Security Architecture

**Recent Hardening (January 2026):**

✅ **SEC-001:** API key hashing upgraded from SHA-256 to Argon2id
- Before: `hash('sha256', $key)` (vulnerable to rainbow tables)
- After: `password_hash($key, PASSWORD_ARGON2ID)` with unique salts
- Side effect: `findByKey()` now iterates active keys (no direct lookup)

✅ **SEC-002:** SQL injection fixed in orderByRaw patterns
- Before: `orderByRaw("FIELD(priority, ...)")`
- After: Parameterised `orderByPriority()` and `orderByStatus()` scopes

✅ **SEC-003:** Workspace scoping added to state tools
- Added `forWorkspace()` scoping to StateSet, StateGet, StateList, PlanGet, PlanList
- Prevents cross-tenant data access

**Outstanding Security Items:**
- ❌ SEC-004: Missing per-tool rate limiting (P1)
- ❌ VAL-001: Template variable injection vulnerability (P1)

**Middleware:**
- `AuthenticateAgent` - API key authentication
- IP whitelist checking via `IpRestrictionService`
- Rate limiting per API key

---

## 3. Code Quality Analysis

### 3.1 Conventions Compliance

✅ **Excellent:**
- UK English throughout (colour, organisation, centre)
- `declare(strict_types=1);` in all PHP files
- Type hints on all parameters and return types
- PSR-12 coding style
- Pest framework for testing
- Conventional commit messages

### 3.2 Documentation Quality

✅ **Very Good:**
- Comprehensive CLAUDE.md with clear guidance
- Well-maintained TODO.md with priority system (P1-P6)
- PHPDoc comments on most methods
- README.md with usage examples
- AGENTS.md for agent-specific instructions
- Changelog tracking (cliff.toml for git-cliff)

### 3.3 Known Issues (from TODO.md)

**Priority 1 (Critical):**
- [ ] SEC-004: Missing per-tool rate limiting
- [ ] VAL-001: Template variable injection vulnerability

**Priority 2 (High):**
- [x] TEST-001 to TEST-005: Test coverage (COMPLETED Jan 2026)
- [x] DB-001: Missing agent_plans migration (COMPLETED Jan 2026)
- [ ] DB-002: Missing indexes on frequently queried columns
- [ ] ERR-001: ClaudeService stream() error handling
- [ ] ERR-002: ContentService batch failure recovery

**Priority 3 (Medium):**
- [ ] DX-001: Unclear workspace context error messages
- [ ] DX-002: AgenticManager no API key validation on init
- [ ] DX-003: Plan template variable errors not actionable
- [ ] CQ-001: Duplicate state models
- [ ] CQ-002: ApiKeyManager uses wrong model
- [ ] CQ-003: Cache key not namespaced
- [ ] PERF-001: N+1 queries in checkDependencies
- [ ] PERF-002: O(n) filter on every request

**Lower Priority:** P4-P6 items documented but not critical

---

## 4. Migration Status

### 4.1 Existing Migrations

```
Migrations/
├── 0001_01_01_000001_create_agentic_tables.php
│   Creates: agent_sessions, agent_api_keys, prompts, prompt_versions, tasks
│
├── 0001_01_01_000002_add_ip_whitelist_to_agent_api_keys.php
│   Adds: ip_whitelist column (JSON)
│
└── 0001_01_01_000003_create_agent_plans_tables.php
    Creates: agent_plans, agent_phases, agent_workspace_states
    Updates: agent_sessions with agent_plan_id FK
```

### 4.2 Idempotency

**Status:** ✅ Recent fix (commit cda896e)

From git log:
```
cda896e fix(migrations): make idempotent and align schemas with models
```

This suggests migrations have been fixed to be safely re-runnable.

---

## 5. Testing Strategy (When Dependencies Resolved)

### 5.1 Recommended Test Execution Order

Once `host-uk/core` dependency is resolved:

```bash
# 1. Install dependencies
composer install --no-interaction

# 2. Run linter
vendor/bin/pint --test
# OR: composer run lint

# 3. Run tests with coverage
vendor/bin/pest --coverage
# OR: composer test

# 4. Run static analysis (if configured)
vendor/bin/phpstan analyse --memory-limit=512M

# 5. Check for security issues
composer audit
```

### 5.2 Test Gaps to Address

**High Priority:**
1. Console commands (TaskCommand, PlanCommand, GenerateCommand)
2. Livewire components (9 admin panel components)
3. MCP tools integration tests
4. Middleware authentication flow

**Medium Priority:**
1. ContentService batch processing
2. Session handoff and resume flows
3. Template variable substitution edge cases
4. Rate limiting behaviour

---

## 6. Key Architectural Patterns

### 6.1 Design Patterns Identified

**1. Service Provider Pattern**
- Event-driven lazy loading via Boot.php
- Modular registration (admin, console, MCP)

**2. Repository Pattern**
- AgentToolRegistry for tool discovery
- PlanTemplateService for template management

**3. Strategy Pattern**
- AgenticProviderInterface for AI providers
- Different providers interchangeable via AgenticManager

**4. Circuit Breaker Pattern**
- Built into AgentTool base class
- Protects against cascading failures

**5. Factory Pattern**
- AgentApiKey::generate() for secure key creation
- Template-based plan creation

### 6.2 SOLID Principles Compliance

✅ **Single Responsibility:** Each service has clear, focused purpose

✅ **Open/Closed:** AgentTool extensible via inheritance, closed for modification

✅ **Liskov Substitution:** All AI providers implement AgenticProviderInterface

✅ **Interface Segregation:** AgentToolInterface, AgenticProviderInterface are minimal

✅ **Dependency Inversion:** Services depend on interfaces, not concrete classes

---

## 7. Recommendations

### 7.1 Immediate Actions (Phase 0 Complete)

1. ✅ Document dependency constraints (this file)
2. ✅ Review architecture and patterns (completed)
3. ✅ Create FINDINGS.md (this file)
4. 🔄 Commit and push to feat/phase-0-assessment
5. 🔄 Comment findings on issue #1

### 7.2 Next Phase Priorities

**Phase 1: Dependency Resolution**
- Add repository configuration for host-uk/core
- OR create mock/stub for isolated testing
- Verify all migrations run successfully

**Phase 2: Test Execution**
- Run full test suite
- Document any test failures
- Check code coverage gaps

**Phase 3: Code Quality**
- Address P1 security issues (SEC-004, VAL-001)
- Add missing indexes (DB-002)
- Fix error handling (ERR-001, ERR-002)

**Phase 4: Documentation**
- Add PHPDoc to undocumented patterns
- Document MCP tool dependency system
- Create integration examples

---

## 8. Conclusions

### 8.1 Overall Assessment

**Grade: B+ (Very Good)**

**Strengths:**
- ✅ Modern, event-driven Laravel architecture
- ✅ Comprehensive test coverage for critical paths
- ✅ Security-conscious with recent hardening
- ✅ Well-documented with clear conventions
- ✅ Clean abstractions and design patterns
- ✅ Excellent TODO.md with prioritised backlog

**Weaknesses:**
- ⚠️ Dependency on private package blocks standalone testing
- ⚠️ Some P1 security items outstanding
- ⚠️ Performance optimisations needed (N+1 queries, caching)
- ⚠️ Test coverage gaps in commands and Livewire

**Risk Assessment:**
- **Security:** MEDIUM (P1 items need attention)
- **Maintainability:** LOW (well-structured, documented)
- **Performance:** LOW-MEDIUM (known issues documented)
- **Testability:** MEDIUM (depends on private package)

### 8.2 Production Readiness

**Current State:** BETA / STAGING-READY

**Blockers for Production:**
1. SEC-004: Per-tool rate limiting
2. VAL-001: Template injection vulnerability
3. ERR-001/002: Error handling in streaming and batch processing
4. DB-002: Missing performance indexes

**Estimate to Production-Ready:** 2-3 sprints

---

## 9. Appendix

### 9.1 File Structure Summary

```
core-agentic/
├── Boot.php                    # Service provider + event listeners
├── composer.json               # Package definition (blocked on host-uk/core)
├── config.php                  # MCP configuration
├── CLAUDE.md                   # Development guidelines
├── TODO.md                     # Prioritised task backlog (12,632 bytes)
├── README.md                   # Package documentation
├── AGENTS.md                   # Agent-specific instructions
│
├── Console/Commands/           # 3 Artisan commands
│   ├── TaskCommand.php
│   ├── PlanCommand.php
│   └── GenerateCommand.php
│
├── Controllers/                # API controllers
│   └── ForAgentsController.php
│
├── Facades/                    # Laravel facades
│
├── Jobs/                       # Queue jobs
│
├── Mcp/                        # Model Context Protocol
│   ├── Prompts/
│   ├── Servers/
│   └── Tools/Agent/            # 34 agent tools
│
├── Middleware/                 # Authentication
│   └── AuthenticateAgent.php
│
├── Migrations/                 # 3 database migrations
│
├── Models/                     # 9 Eloquent models
│
├── routes/                     # API and admin routes
│
├── Service/                    # Legacy namespace?
│
├── Services/                   # 15 service classes
│   ├── AgenticManager.php      # AI provider coordinator
│   ├── *Service.php            # Domain services
│   └── Concerns/               # Shared traits
│
├── Support/                    # Helper utilities
│
├── tests/                      # Pest test suite
│   ├── Feature/                # 9 feature tests
│   ├── Unit/                   # 4 unit tests
│   ├── UseCase/                # 1 use case test
│   ├── Pest.php                # Test configuration
│   └── TestCase.php            # Base test class
│
└── View/                       # UI components
    ├── Blade/admin/            # Admin panel views
    └── Modal/Admin/            # 9 Livewire components
```

### 9.2 Dependencies (from composer.json)

**Runtime:**
- PHP ^8.2
- host-uk/core dev-main (PRIVATE - blocks installation)

**Development:**
- laravel/pint ^1.18 (code formatting)
- orchestra/testbench ^9.0|^10.0 (testing)
- pestphp/pest ^3.0 (testing)

**Note:** PHPStan not listed in composer.json despite TODO.md mentioning it

### 9.3 Git Status

```
Branch: feat/phase-0-assessment (created from main)
Status: Clean working directory
Recent commits on main:
  cda896e fix(migrations): make idempotent and align schemas with models
  c439194 feat(menu): move Agentic to dedicated agents group
  bf7c0d7 fix(models): add context array cast to AgentPlan
```

---

**Assessment Completed:** 2026-02-20
**Next Action:** Commit findings and comment on issue #1
