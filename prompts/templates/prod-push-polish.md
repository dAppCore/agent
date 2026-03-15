# Production Push Polish Template

**Use when:** Preparing a codebase for production deployment after feature development is complete.

**Purpose:** Ensure all routes work correctly, render meaningful content, handle errors gracefully, and meet security/performance standards.

---

## How to Request This Task

When asking an agent to create a prod push polish task, include:

```
Create a production push polish task following the template at:
resources/plan-templates/prod-push-polish.md

Focus areas: [list any specific concerns]
Target deployment date: [date if applicable]
```

---

## Task Structure

### Phase 1: Public Route Tests

Every public route must have a test that:
1. Asserts HTTP 200 OK status
2. Asserts meaningful HTML content renders (title, headings, key elements)
3. Does NOT just use `assertOk()` alone

**Pattern:**
```php
it('renders [page name] with [key content]', function () {
    $this->get('/route')
        ->assertOk()
        ->assertSee('Expected heading')
        ->assertSee('Expected content')
        ->assertSee('Expected CTA');
});
```

**Why:** A page can return 200 with blank body, PHP errors, or broken layout. Content assertions catch these.

### Phase 2: Authenticated Route Tests

Every authenticated route must have a test that:
1. Uses `actingAs()` with appropriate user type
2. Asserts the Livewire component renders
3. Asserts key UI elements are present

**Pattern:**
```php
it('renders [page name] for authenticated user', function () {
    $this->actingAs($this->user)
        ->get('/hub/route')
        ->assertOk()
        ->assertSeeLivewire('component.name')
        ->assertSee('Expected heading')
        ->assertSee('Expected widget');
});
```

### Phase 3: Error Page Verification

Error pages must:
1. Use consistent brand styling (not default Laravel)
2. Provide helpful messages
3. Include navigation back to safe pages
4. Not expose stack traces in production

**Test pattern:**
```php
it('renders 404 with helpful message', function () {
    $this->get('/nonexistent-route')
        ->assertNotFound()
        ->assertSee('Page not found')
        ->assertSee('Go to homepage')
        ->assertDontSee('Exception');
});
```

### Phase 4: Security Headers

Verify these headers are present on all responses:

| Header | Value | Purpose |
|--------|-------|---------|
| `X-Frame-Options` | `DENY` or `SAMEORIGIN` | Prevent clickjacking |
| `X-Content-Type-Options` | `nosniff` | Prevent MIME sniffing |
| `Referrer-Policy` | `strict-origin-when-cross-origin` | Control referrer info |
| `Content-Security-Policy` | (varies) | Prevent XSS |
| `X-Powered-By` | (removed) | Don't expose stack |

**Middleware pattern:**
```php
class SecurityHeaders
{
    public function handle(Request $request, Closure $next): Response
    {
        $response = $next($request);

        $response->headers->set('X-Frame-Options', 'DENY');
        $response->headers->set('X-Content-Type-Options', 'nosniff');
        $response->headers->set('Referrer-Policy', 'strict-origin-when-cross-origin');
        $response->headers->remove('X-Powered-By');

        return $response;
    }
}
```

### Phase 5: Performance Baseline

Document response times for key routes:

| Route Type | Target | Acceptable | Needs Investigation |
|------------|--------|------------|---------------------|
| Static marketing | <200ms | <400ms | >600ms |
| Dynamic public | <300ms | <500ms | >800ms |
| Authenticated dashboard | <500ms | <800ms | >1200ms |
| Data-heavy pages | <800ms | <1200ms | >2000ms |

**Test pattern:**
```php
it('responds within performance target', function () {
    $start = microtime(true);

    $this->get('/');

    $duration = (microtime(true) - $start) * 1000;

    expect($duration)->toBeLessThan(400); // ms
});
```

**N+1 detection:**
```php
it('has no N+1 queries on listing page', function () {
    DB::enableQueryLog();

    $this->get('/hub/social/posts');

    $queries = DB::getQueryLog();

    // With 10 posts, should be ~3 queries (posts, accounts, user)
    // not 10+ (one per post)
    expect(count($queries))->toBeLessThan(10);
});
```

### Phase 6: Final Verification

Pre-deployment checklist:

- [ ] `./vendor/bin/pest` — 0 failures
- [ ] `npm run build` — 0 errors
- [ ] `npm run test:smoke` — Playwright passes
- [ ] Error pages reviewed manually
- [ ] Security headers verified via browser dev tools
- [ ] Performance baselines documented

---

## Acceptance Criteria Template

Copy and customise for your task:

```markdown
### Phase 1: Public Route Tests
- [ ] AC1: Test for `/` asserts page title and hero content
- [ ] AC2: Test for `/pricing` asserts pricing tiers display
- [ ] AC3: Test for `/login` asserts form fields render
[Add one AC per route]

### Phase 2: Authenticated Route Tests
- [ ] AC4: Test for `/hub` asserts dashboard widgets render
- [ ] AC5: Test for `/hub/profile` asserts form with user data
[Add one AC per authenticated route]

### Phase 3: Error Page Verification
- [ ] AC6: 404 page renders with brand styling
- [ ] AC7: 403 page renders with access denied message
- [ ] AC8: 500 page renders without stack trace

### Phase 4: Security Headers
- [ ] AC9: X-Frame-Options header present
- [ ] AC10: X-Content-Type-Options header present
- [ ] AC11: Referrer-Policy header present
- [ ] AC12: X-Powered-By header removed

### Phase 5: Performance Baseline
- [ ] AC13: Homepage <400ms response time
- [ ] AC14: No N+1 queries on listing pages
- [ ] AC15: Performance baselines documented

### Phase 6: Final Verification
- [ ] AC16: Full test suite passes
- [ ] AC17: Build completes without errors
- [ ] AC18: Smoke tests pass
```

---

## Common Issues to Check

### Blank Pages
- Missing `return` in controller
- Livewire component `render()` returns nothing
- Blade `@extends` pointing to missing layout

### Broken Layouts
- Missing Flux UI components
- CSS not loaded (Vite build issue)
- Alpine.js not initialised

### Authentication Redirects
- Middleware order incorrect
- Session not persisting
- CSRF token mismatch

### Performance Problems
- Eager loading missing (N+1)
- Large datasets not paginated
- Expensive queries in loops
- Missing database indexes

### Security Gaps
- Debug mode enabled in production
- Sensitive data in logs
- Missing CSRF protection
- Exposed .env or config values

---

## File Locations

| Purpose | Location |
|---------|----------|
| Route tests | `tests/Feature/PublicRoutesTest.php`, `tests/Feature/HubRoutesTest.php` |
| Error pages | `resources/views/errors/` |
| Security middleware | `app/Http/Middleware/SecurityHeaders.php` |
| Performance tests | `tests/Feature/PerformanceBaselineTest.php` |
| Smoke tests | `tests/Browser/smoke/` |

---

*This template ensures production deployments don't ship broken pages.*
