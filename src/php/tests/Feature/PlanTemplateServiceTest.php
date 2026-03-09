<?php

declare(strict_types=1);

/**
 * Tests for the PlanTemplateService.
 *
 * Covers template loading, variable substitution, plan creation, and validation.
 */

use Core\Mod\Agentic\Models\AgentPlan;
use Core\Mod\Agentic\Services\PlanTemplateService;
use Core\Tenant\Models\Workspace;
use Illuminate\Support\Facades\File;
use Symfony\Component\Yaml\Yaml;

// =========================================================================
// Test Setup
// =========================================================================

beforeEach(function () {
    $this->workspace = Workspace::factory()->create();
    $this->service = app(PlanTemplateService::class);
    $this->testTemplatesPath = resource_path('plan-templates');

    if (! File::isDirectory($this->testTemplatesPath)) {
        File::makeDirectory($this->testTemplatesPath, 0755, true);
    }
});

afterEach(function () {
    if (File::isDirectory($this->testTemplatesPath)) {
        File::deleteDirectory($this->testTemplatesPath);
    }
});

/**
 * Create a test template file.
 */
function createTestTemplate(string $slug, array $content): void
{
    $path = resource_path('plan-templates');
    $yaml = Yaml::dump($content, 10);
    File::put($path.'/'.$slug.'.yaml', $yaml);
}

// =========================================================================
// Template Listing Tests
// =========================================================================

describe('template listing', function () {
    it('returns empty collection when no templates exist', function () {
        File::cleanDirectory($this->testTemplatesPath);

        $result = $this->service->list();

        expect($result->isEmpty())->toBeTrue();
    });

    it('returns templates sorted by name', function () {
        createTestTemplate('zebra-template', ['name' => 'Zebra Template', 'phases' => []]);
        createTestTemplate('alpha-template', ['name' => 'Alpha Template', 'phases' => []]);
        createTestTemplate('middle-template', ['name' => 'Middle Template', 'phases' => []]);

        $result = $this->service->list();

        expect($result)->toHaveCount(3)
            ->and($result[0]['name'])->toBe('Alpha Template')
            ->and($result[1]['name'])->toBe('Middle Template')
            ->and($result[2]['name'])->toBe('Zebra Template');
    });

    it('includes template metadata', function () {
        createTestTemplate('test-template', [
            'name' => 'Test Template',
            'description' => 'A test description',
            'category' => 'testing',
            'phases' => [
                ['name' => 'Phase 1', 'tasks' => ['Task 1']],
                ['name' => 'Phase 2', 'tasks' => ['Task 2']],
            ],
        ]);

        $result = $this->service->list();

        expect($result)->toHaveCount(1);

        $template = $result[0];
        expect($template['slug'])->toBe('test-template')
            ->and($template['name'])->toBe('Test Template')
            ->and($template['description'])->toBe('A test description')
            ->and($template['category'])->toBe('testing')
            ->and($template['phases_count'])->toBe(2);
    });

    it('extracts variable definitions', function () {
        createTestTemplate('with-vars', [
            'name' => 'Template with Variables',
            'variables' => [
                'project_name' => [
                    'description' => 'The project name',
                    'required' => true,
                ],
                'author' => [
                    'description' => 'Author name',
                    'default' => 'Anonymous',
                    'required' => false,
                ],
            ],
            'phases' => [],
        ]);

        $result = $this->service->list();
        $template = $result[0];

        expect($template['variables'])->toHaveCount(2)
            ->and($template['variables'][0]['name'])->toBe('project_name')
            ->and($template['variables'][0]['required'])->toBeTrue()
            ->and($template['variables'][1]['name'])->toBe('author')
            ->and($template['variables'][1]['default'])->toBe('Anonymous');
    });

    it('ignores non-YAML files', function () {
        createTestTemplate('valid-template', ['name' => 'Valid', 'phases' => []]);
        File::put($this->testTemplatesPath.'/readme.txt', 'Not a template');
        File::put($this->testTemplatesPath.'/config.json', '{}');

        $result = $this->service->list();

        expect($result)->toHaveCount(1)
            ->and($result[0]['slug'])->toBe('valid-template');
    });

    it('returns array from listTemplates method', function () {
        createTestTemplate('test-template', ['name' => 'Test', 'phases' => []]);

        $result = $this->service->listTemplates();

        expect($result)->toBeArray()
            ->toHaveCount(1);
    });
});

// =========================================================================
// Get Template Tests
// =========================================================================

describe('template retrieval', function () {
    it('returns template content by slug', function () {
        createTestTemplate('my-template', [
            'name' => 'My Template',
            'description' => 'Test description',
            'phases' => [
                ['name' => 'Phase 1', 'tasks' => ['Task A']],
            ],
        ]);

        $result = $this->service->get('my-template');

        expect($result)->not->toBeNull()
            ->and($result['slug'])->toBe('my-template')
            ->and($result['name'])->toBe('My Template')
            ->and($result['description'])->toBe('Test description');
    });

    it('returns null for nonexistent template', function () {
        $result = $this->service->get('nonexistent-template');

        expect($result)->toBeNull();
    });

    it('supports .yml extension', function () {
        $yaml = Yaml::dump(['name' => 'YML Template', 'phases' => []], 10);
        File::put($this->testTemplatesPath.'/yml-template.yml', $yaml);

        $result = $this->service->get('yml-template');

        expect($result)->not->toBeNull()
            ->and($result['name'])->toBe('YML Template');
    });
});

// =========================================================================
// Preview Template Tests
// =========================================================================

describe('template preview', function () {
    it('returns complete preview structure', function () {
        createTestTemplate('preview-test', [
            'name' => 'Preview Test',
            'description' => 'Testing preview',
            'category' => 'test',
            'phases' => [
                ['name' => 'Setup', 'description' => 'Initial setup', 'tasks' => ['Install deps']],
            ],
            'guidelines' => ['Be thorough', 'Test everything'],
        ]);

        $result = $this->service->previewTemplate('preview-test');

        expect($result)->not->toBeNull()
            ->and($result['slug'])->toBe('preview-test')
            ->and($result['name'])->toBe('Preview Test')
            ->and($result['description'])->toBe('Testing preview')
            ->and($result['category'])->toBe('test')
            ->and($result['phases'])->toHaveCount(1)
            ->and($result['phases'][0]['order'])->toBe(1)
            ->and($result['phases'][0]['name'])->toBe('Setup')
            ->and($result['guidelines'])->toHaveCount(2);
    });

    it('returns null for nonexistent template', function () {
        $result = $this->service->previewTemplate('nonexistent');

        expect($result)->toBeNull();
    });

    it('applies variable substitution', function () {
        createTestTemplate('var-preview', [
            'name' => '{{ project_name }} Plan',
            'description' => 'Plan for {{ project_name }}',
            'phases' => [
                ['name' => 'Work on {{ project_name }}', 'tasks' => ['{{ task_type }}']],
            ],
        ]);

        $result = $this->service->previewTemplate('var-preview', [
            'project_name' => 'MyProject',
            'task_type' => 'Build feature',
        ]);

        expect($result['name'])->toContain('MyProject')
            ->and($result['description'])->toContain('MyProject')
            ->and($result['phases'][0]['name'])->toContain('MyProject');
    });

    it('includes applied variables in response', function () {
        createTestTemplate('track-vars', [
            'name' => '{{ name }} Template',
            'phases' => [],
        ]);

        $result = $this->service->previewTemplate('track-vars', ['name' => 'Test']);

        expect($result)->toHaveKey('variables_applied')
            ->and($result['variables_applied'])->toBe(['name' => 'Test']);
    });
});

// =========================================================================
// Variable Substitution Tests
// =========================================================================

describe('variable substitution', function () {
    it('substitutes simple variables', function () {
        createTestTemplate('simple-vars', [
            'name' => '{{ project }} Project',
            'phases' => [],
        ]);

        $result = $this->service->previewTemplate('simple-vars', ['project' => 'Alpha']);

        expect($result['name'])->toBe('Alpha Project');
    });

    it('handles whitespace in variable placeholders', function () {
        createTestTemplate('whitespace-vars', [
            'name' => '{{  project  }} Project',
            'phases' => [],
        ]);

        $result = $this->service->previewTemplate('whitespace-vars', ['project' => 'Beta']);

        expect($result['name'])->toBe('Beta Project');
    });

    it('applies default values when variable not provided', function () {
        createTestTemplate('default-vars', [
            'name' => '{{ project }} by {{ author }}',
            'variables' => [
                'project' => ['required' => true],
                'author' => ['default' => 'Unknown'],
            ],
            'phases' => [],
        ]);

        $result = $this->service->previewTemplate('default-vars', ['project' => 'Gamma']);

        expect($result['name'])->toBe('Gamma by Unknown');
    });

    it('handles special characters in variable values', function () {
        createTestTemplate('special-chars', [
            'name' => '{{ title }}',
            'description' => '{{ desc }}',
            'phases' => [],
        ]);

        $result = $this->service->previewTemplate('special-chars', [
            'title' => 'Test "quotes" and \\backslashes\\',
            'desc' => 'Has <html> & "quotes"',
        ]);

        expect($result)->not->toBeNull()
            ->and($result['name'])->toContain('quotes');
    });

    it('ignores non-scalar variable values', function () {
        createTestTemplate('scalar-only', [
            'name' => '{{ project }}',
            'phases' => [],
        ]);

        $result = $this->service->previewTemplate('scalar-only', [
            'project' => ['array' => 'value'],
        ]);

        expect($result['name'])->toContain('{{ project }}');
    });

    it('handles multiple occurrences of same variable', function () {
        createTestTemplate('multi-occurrence', [
            'name' => '{{ app }} - {{ app }}',
            'description' => 'This is {{ app }}',
            'phases' => [],
        ]);

        $result = $this->service->previewTemplate('multi-occurrence', ['app' => 'TestApp']);

        expect($result['name'])->toBe('TestApp - TestApp')
            ->and($result['description'])->toBe('This is TestApp');
    });

    it('preserves unsubstituted variables when value not provided', function () {
        createTestTemplate('unsubstituted', [
            'name' => '{{ provided }} and {{ missing }}',
            'phases' => [],
        ]);

        $result = $this->service->previewTemplate('unsubstituted', ['provided' => 'Here']);

        expect($result['name'])->toBe('Here and {{ missing }}');
    });
});

// =========================================================================
// Create Plan Tests
// =========================================================================

describe('plan creation from template', function () {
    it('creates plan with correct attributes', function () {
        createTestTemplate('create-test', [
            'name' => 'Test Template',
            'description' => 'Template description',
            'phases' => [
                ['name' => 'Phase 1', 'tasks' => ['Task 1', 'Task 2']],
                ['name' => 'Phase 2', 'tasks' => ['Task 3']],
            ],
        ]);

        $plan = $this->service->createPlan('create-test', [], [], $this->workspace);

        expect($plan)->not->toBeNull()
            ->toBeInstanceOf(AgentPlan::class)
            ->and($plan->title)->toBe('Test Template')
            ->and($plan->description)->toBe('Template description')
            ->and($plan->workspace_id)->toBe($this->workspace->id)
            ->and($plan->agentPhases)->toHaveCount(2);
    });

    it('returns null for nonexistent template', function () {
        $result = $this->service->createPlan('nonexistent', [], [], $this->workspace);

        expect($result)->toBeNull();
    });

    it('uses custom title when provided', function () {
        createTestTemplate('custom-title', [
            'name' => 'Template Name',
            'phases' => [],
        ]);

        $plan = $this->service->createPlan(
            'custom-title',
            [],
            ['title' => 'My Custom Title'],
            $this->workspace
        );

        expect($plan->title)->toBe('My Custom Title');
    });

    it('uses custom slug when provided', function () {
        createTestTemplate('custom-slug', [
            'name' => 'Template',
            'phases' => [],
        ]);

        $plan = $this->service->createPlan(
            'custom-slug',
            [],
            ['slug' => 'my-custom-slug'],
            $this->workspace
        );

        expect($plan->slug)->toBe('my-custom-slug');
    });

    it('applies variables to plan content', function () {
        createTestTemplate('var-plan', [
            'name' => '{{ project }} Plan',
            'description' => 'Plan for {{ project }}',
            'phases' => [
                ['name' => '{{ project }} Setup', 'tasks' => ['Configure {{ project }}']],
            ],
        ]);

        $plan = $this->service->createPlan(
            'var-plan',
            ['project' => 'MyApp'],
            [],
            $this->workspace
        );

        expect($plan->title)->toContain('MyApp')
            ->and($plan->description)->toContain('MyApp')
            ->and($plan->agentPhases[0]->name)->toContain('MyApp');
    });

    it('activates plan when requested', function () {
        createTestTemplate('activate-plan', [
            'name' => 'Activatable',
            'phases' => [],
        ]);

        $plan = $this->service->createPlan(
            'activate-plan',
            [],
            ['activate' => true],
            $this->workspace
        );

        expect($plan->status)->toBe(AgentPlan::STATUS_ACTIVE);
    });

    it('defaults to draft status', function () {
        createTestTemplate('draft-plan', [
            'name' => 'Draft Plan',
            'phases' => [],
        ]);

        $plan = $this->service->createPlan('draft-plan', [], [], $this->workspace);

        expect($plan->status)->toBe(AgentPlan::STATUS_DRAFT);
    });

    it('stores template metadata', function () {
        createTestTemplate('metadata-plan', [
            'name' => 'Metadata Template',
            'phases' => [],
        ]);

        $plan = $this->service->createPlan(
            'metadata-plan',
            ['var1' => 'value1'],
            [],
            $this->workspace
        );

        expect($plan->metadata['source'])->toBe('template')
            ->and($plan->metadata['template_slug'])->toBe('metadata-plan')
            ->and($plan->metadata['variables'])->toBe(['var1' => 'value1']);
    });

    it('creates phases in correct order', function () {
        createTestTemplate('ordered-phases', [
            'name' => 'Ordered',
            'phases' => [
                ['name' => 'First'],
                ['name' => 'Second'],
                ['name' => 'Third'],
            ],
        ]);

        $plan = $this->service->createPlan('ordered-phases', [], [], $this->workspace);

        expect($plan->agentPhases[0]->order)->toBe(1)
            ->and($plan->agentPhases[0]->name)->toBe('First')
            ->and($plan->agentPhases[1]->order)->toBe(2)
            ->and($plan->agentPhases[1]->name)->toBe('Second')
            ->and($plan->agentPhases[2]->order)->toBe(3)
            ->and($plan->agentPhases[2]->name)->toBe('Third');
    });

    it('creates tasks with pending status', function () {
        createTestTemplate('task-status', [
            'name' => 'Task Status',
            'phases' => [
                ['name' => 'Phase', 'tasks' => ['Task 1', 'Task 2']],
            ],
        ]);

        $plan = $this->service->createPlan('task-status', [], [], $this->workspace);
        $tasks = $plan->agentPhases[0]->tasks;

        expect($tasks[0]['status'])->toBe('pending')
            ->and($tasks[1]['status'])->toBe('pending');
    });

    it('handles complex task definitions', function () {
        createTestTemplate('complex-tasks', [
            'name' => 'Complex Tasks',
            'phases' => [
                [
                    'name' => 'Phase',
                    'tasks' => [
                        ['name' => 'Simple task'],
                        ['name' => 'Task with metadata', 'priority' => 'high'],
                    ],
                ],
            ],
        ]);

        $plan = $this->service->createPlan('complex-tasks', [], [], $this->workspace);
        $tasks = $plan->agentPhases[0]->tasks;

        expect($tasks[0]['name'])->toBe('Simple task')
            ->and($tasks[1]['name'])->toBe('Task with metadata')
            ->and($tasks[1]['priority'])->toBe('high');
    });

    it('accepts workspace_id via options', function () {
        createTestTemplate('workspace-id-option', [
            'name' => 'Test',
            'phases' => [],
        ]);

        $plan = $this->service->createPlan(
            'workspace-id-option',
            [],
            ['workspace_id' => $this->workspace->id]
        );

        expect($plan->workspace_id)->toBe($this->workspace->id);
    });

    it('creates plan without workspace when none provided', function () {
        createTestTemplate('no-workspace', [
            'name' => 'No Workspace',
            'phases' => [],
        ]);

        $plan = $this->service->createPlan('no-workspace', [], []);

        expect($plan->workspace_id)->toBeNull();
    });
});

// =========================================================================
// Variable Validation Tests
// =========================================================================

describe('variable validation', function () {
    it('returns valid when all required variables provided', function () {
        createTestTemplate('validate-vars', [
            'name' => 'Test',
            'variables' => [
                'required_var' => ['required' => true],
            ],
            'phases' => [],
        ]);

        $result = $this->service->validateVariables('validate-vars', ['required_var' => 'value']);

        expect($result['valid'])->toBeTrue()
            ->and($result['errors'])->toBeEmpty();
    });

    it('returns error when required variable missing', function () {
        createTestTemplate('missing-required', [
            'name' => 'Test',
            'variables' => [
                'required_var' => ['required' => true],
            ],
            'phases' => [],
        ]);

        $result = $this->service->validateVariables('missing-required', []);

        expect($result['valid'])->toBeFalse()
            ->and($result['errors'])->not->toBeEmpty()
            ->and($result['errors'][0])->toContain('required_var');
    });

    it('accepts default value for required variable', function () {
        createTestTemplate('default-required', [
            'name' => 'Test',
            'variables' => [
                'optional_with_default' => ['required' => true, 'default' => 'default value'],
            ],
            'phases' => [],
        ]);

        $result = $this->service->validateVariables('default-required', []);

        expect($result['valid'])->toBeTrue();
    });

    it('returns error for nonexistent template', function () {
        $result = $this->service->validateVariables('nonexistent', []);

        expect($result['valid'])->toBeFalse()
            ->and($result['errors'][0])->toContain('Template not found');
    });

    it('validates multiple required variables', function () {
        createTestTemplate('multi-required', [
            'name' => 'Test',
            'variables' => [
                'var1' => ['required' => true],
                'var2' => ['required' => true],
                'var3' => ['required' => false],
            ],
            'phases' => [],
        ]);

        $result = $this->service->validateVariables('multi-required', ['var1' => 'a']);

        expect($result['valid'])->toBeFalse()
            ->and($result['errors'])->toHaveCount(1)
            ->and($result['errors'][0])->toContain('var2');
    });

    it('includes description in error message', function () {
        createTestTemplate('desc-in-error', [
            'name' => 'Test',
            'variables' => [
                'project_name' => [
                    'required' => true,
                    'description' => 'The name of the project being planned',
                ],
            ],
            'phases' => [],
        ]);

        $result = $this->service->validateVariables('desc-in-error', []);

        expect($result['errors'][0])
            ->toContain('project_name')
            ->toContain('The name of the project being planned');
    });

    it('includes example value in error message', function () {
        createTestTemplate('example-in-error', [
            'name' => 'Test',
            'variables' => [
                'api_key' => [
                    'required' => true,
                    'description' => 'Your API key',
                    'example' => 'sk-abc123',
                ],
            ],
            'phases' => [],
        ]);

        $result = $this->service->validateVariables('example-in-error', []);

        expect($result['errors'][0])
            ->toContain('api_key')
            ->toContain('sk-abc123');
    });

    it('includes multiple examples in error message', function () {
        createTestTemplate('examples-list-in-error', [
            'name' => 'Test',
            'variables' => [
                'environment' => [
                    'required' => true,
                    'description' => 'Deployment environment',
                    'examples' => ['production', 'staging', 'development'],
                ],
            ],
            'phases' => [],
        ]);

        $result = $this->service->validateVariables('examples-list-in-error', []);

        expect($result['errors'][0])
            ->toContain('environment')
            ->toContain('production')
            ->toContain('staging');
    });

    it('includes expected format in error message', function () {
        createTestTemplate('format-in-error', [
            'name' => 'Test',
            'variables' => [
                'release_date' => [
                    'required' => true,
                    'description' => 'Date of the planned release',
                    'format' => 'YYYY-MM-DD',
                    'example' => '2026-03-01',
                ],
            ],
            'phases' => [],
        ]);

        $result = $this->service->validateVariables('format-in-error', []);

        expect($result['errors'][0])
            ->toContain('release_date')
            ->toContain('YYYY-MM-DD')
            ->toContain('2026-03-01');
    });

    it('returns naming convention in all results', function () {
        createTestTemplate('naming-convention', [
            'name' => 'Test',
            'variables' => [
                'my_var' => ['required' => true],
            ],
            'phases' => [],
        ]);

        $invalid = $this->service->validateVariables('naming-convention', []);
        $valid = $this->service->validateVariables('naming-convention', ['my_var' => 'value']);

        expect($invalid)->toHaveKey('naming_convention')
            ->and($invalid['naming_convention'])->toContain('snake_case')
            ->and($valid)->toHaveKey('naming_convention')
            ->and($valid['naming_convention'])->toContain('snake_case');
    });

    it('error message without description is still actionable', function () {
        createTestTemplate('no-desc-error', [
            'name' => 'Test',
            'variables' => [
                'bare_var' => ['required' => true],
            ],
            'phases' => [],
        ]);

        $result = $this->service->validateVariables('no-desc-error', []);

        expect($result['errors'][0])
            ->toContain('bare_var')
            ->toContain('missing');
    });
});

// =========================================================================
// Category Tests
// =========================================================================

describe('template categories', function () {
    it('filters templates by category', function () {
        createTestTemplate('dev-1', ['name' => 'Dev 1', 'category' => 'development', 'phases' => []]);
        createTestTemplate('dev-2', ['name' => 'Dev 2', 'category' => 'development', 'phases' => []]);
        createTestTemplate('ops-1', ['name' => 'Ops 1', 'category' => 'operations', 'phases' => []]);

        $devTemplates = $this->service->getByCategory('development');

        expect($devTemplates)->toHaveCount(2);
        foreach ($devTemplates as $template) {
            expect($template['category'])->toBe('development');
        }
    });

    it('returns unique categories sorted alphabetically', function () {
        createTestTemplate('t1', ['name' => 'T1', 'category' => 'alpha', 'phases' => []]);
        createTestTemplate('t2', ['name' => 'T2', 'category' => 'beta', 'phases' => []]);
        createTestTemplate('t3', ['name' => 'T3', 'category' => 'alpha', 'phases' => []]);

        $categories = $this->service->getCategories();

        expect($categories)->toHaveCount(2)
            ->and($categories->toArray())->toContain('alpha')
            ->and($categories->toArray())->toContain('beta');
    });

    it('returns categories in sorted order', function () {
        createTestTemplate('t1', ['name' => 'T1', 'category' => 'zebra', 'phases' => []]);
        createTestTemplate('t2', ['name' => 'T2', 'category' => 'alpha', 'phases' => []]);

        $categories = $this->service->getCategories();

        expect($categories[0])->toBe('alpha')
            ->and($categories[1])->toBe('zebra');
    });

    it('returns empty collection when no templates', function () {
        File::cleanDirectory($this->testTemplatesPath);

        $categories = $this->service->getCategories();

        expect($categories)->toBeEmpty();
    });
});

// =========================================================================
// Context Building Tests
// =========================================================================

describe('context generation', function () {
    it('builds context from template data', function () {
        createTestTemplate('with-context', [
            'name' => 'Context Test',
            'description' => 'Testing context generation',
            'guidelines' => ['Guideline 1', 'Guideline 2'],
            'phases' => [],
        ]);

        $plan = $this->service->createPlan(
            'with-context',
            ['project' => 'TestProject'],
            [],
            $this->workspace
        );

        expect($plan->context)->not->toBeNull()
            ->toContain('Context Test')
            ->toContain('Testing context generation')
            ->toContain('Guideline 1');
    });

    it('uses explicit context when provided in template', function () {
        createTestTemplate('explicit-context', [
            'name' => 'Test',
            'context' => 'This is the explicit context.',
            'phases' => [],
        ]);

        $plan = $this->service->createPlan('explicit-context', [], [], $this->workspace);

        expect($plan->context)->toBe('This is the explicit context.');
    });

    it('includes variables in generated context', function () {
        createTestTemplate('vars-in-context', [
            'name' => 'Variable Context',
            'description' => 'A plan with variables',
            'phases' => [],
        ]);

        $plan = $this->service->createPlan(
            'vars-in-context',
            ['key1' => 'value1', 'key2' => 'value2'],
            [],
            $this->workspace
        );

        expect($plan->context)
            ->toContain('key1')
            ->toContain('value1')
            ->toContain('key2')
            ->toContain('value2');
    });
});

// =========================================================================
// Edge Cases and Error Handling
// =========================================================================

describe('edge cases', function () {
    it('handles empty phases array', function () {
        createTestTemplate('no-phases', [
            'name' => 'No Phases',
            'phases' => [],
        ]);

        $plan = $this->service->createPlan('no-phases', [], [], $this->workspace);

        expect($plan->agentPhases)->toBeEmpty();
    });

    it('handles phases without tasks', function () {
        createTestTemplate('no-tasks', [
            'name' => 'No Tasks',
            'phases' => [
                ['name' => 'Empty Phase'],
            ],
        ]);

        $plan = $this->service->createPlan('no-tasks', [], [], $this->workspace);

        expect($plan->agentPhases)->toHaveCount(1)
            ->and($plan->agentPhases[0]->tasks)->toBeEmpty();
    });

    it('handles template without description', function () {
        createTestTemplate('no-desc', [
            'name' => 'No Description',
            'phases' => [],
        ]);

        $plan = $this->service->createPlan('no-desc', [], [], $this->workspace);

        expect($plan->description)->toBeNull();
    });

    it('handles template without variables section', function () {
        createTestTemplate('no-vars-section', [
            'name' => 'No Variables',
            'phases' => [],
        ]);

        $result = $this->service->validateVariables('no-vars-section', []);

        expect($result['valid'])->toBeTrue();
    });

    it('handles malformed YAML gracefully', function () {
        File::put($this->testTemplatesPath.'/malformed.yaml', 'invalid: yaml: content: [');

        // Should not throw when listing
        $result = $this->service->list();

        // Malformed template may be excluded or cause specific behaviour
        expect($result)->toBeInstanceOf(\Illuminate\Support\Collection::class);
    });

    it('generates unique slug for plans with same title', function () {
        createTestTemplate('duplicate-title', [
            'name' => 'Same Title',
            'phases' => [],
        ]);

        $plan1 = $this->service->createPlan('duplicate-title', [], [], $this->workspace);
        $plan2 = $this->service->createPlan('duplicate-title', [], [], $this->workspace);

        expect($plan1->slug)->not->toBe($plan2->slug);
    });
});
