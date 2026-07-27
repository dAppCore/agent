<!-- SPDX-License-Identifier: EUPL-1.2 -->
# Testing

## Go tests

```bash
cd go
go test ./... -count=1                       # all
go test ./pkg/agentic/ -run TestDispatch_Good # one
go vet ./...
GOWORK=off go test ./... -count=1            # CI parity
```

Tests use `testify/assert` and `testify/require`, one test file per source file. Naming is
`TestFilename_FunctionName_<Category>`:

| Suffix | Meaning |
|--------|---------|
| `_Good` | happy path — prove the contract works |
| `_Bad` | expected error conditions |
| `_Ugly` | panics and edge cases |

The suite is substantial — hundreds of tests across `agentic`, `brain`, `lemma`,
`monitor`, `runner`, `setup`. Each `*_example_test.go` doubles as a runnable usage example.

## PHP tests

```bash
composer test                                 # full Pest suite
./vendor/bin/pest --filter=AgenticManagerTest # one file
composer lint                                 # fix code style
```

Pest + Orchestra Testbench. Feature tests use `RefreshDatabase`. Config in
`src/php/tests/Pest.php`:

```php
uses(TestCase::class)->in('Feature', 'Unit', 'UseCase');
uses(RefreshDatabase::class)->in('Feature');
```

Helpers: `createWorkspace()`, `createApiKey($workspace, 'Test Key', ['plan:read'], 100)`.
Suites cover Unit (provider services, manager, detection), Feature (plans/phases/sessions,
API keys, Forgejo, security), Livewire (admin components), and UseCase.

## Conventions

### Go

```go
func TestMyFunction_Good(t *testing.T) {
    result := MyFunction("valid")
    assert.Equal(t, "expected", result)
}
func TestMyFunction_Bad_EmptyInput(t *testing.T) {
    _, err := MyFunction("")
    require.Error(t, err)
    assert.Contains(t, err.Error(), "input required")
}
func TestMyFunction_Ugly_NilPointer(t *testing.T) {
    assert.Panics(t, func() { MyFunction(nil) })
}
```

Use `require` for preconditions (stops the test), `assert` for verifications (reports all).

### PHP (Pest)

```php
it('creates a plan with phases', function () {
    $workspace = createWorkspace();
    $plan = AgentPlan::factory()->create(['workspace_id' => $workspace->id]);
    expect($plan->workspace_id)->toBe($workspace->id);
});
```

Feature tests get `RefreshDatabase` automatically; unit tests should not touch the database.
