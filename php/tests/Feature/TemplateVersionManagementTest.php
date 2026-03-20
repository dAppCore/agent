<?php

declare(strict_types=1);

/**
 * Tests for template version management (FEAT-003).
 *
 * Covers:
 * - Version snapshot creation when a plan is created from a template
 * - Deduplication: identical template content reuses the same version row
 * - New versions are created when template content changes
 * - Plans reference their template version
 * - Version history and retrieval via PlanTemplateService
 */

use Core\Mod\Agentic\Models\AgentPlan;
use Core\Mod\Agentic\Models\PlanTemplateVersion;
use Core\Mod\Agentic\Services\PlanTemplateService;
use Core\Tenant\Models\Workspace;
use Illuminate\Support\Facades\File;
use Symfony\Component\Yaml\Yaml;

// =========================================================================
// Setup helpers
// =========================================================================

beforeEach(function () {
    $this->workspace = Workspace::factory()->create();
    $this->service = app(PlanTemplateService::class);
    $this->templatesPath = resource_path('plan-templates');

    if (! File::isDirectory($this->templatesPath)) {
        File::makeDirectory($this->templatesPath, 0755, true);
    }
});

afterEach(function () {
    if (File::isDirectory($this->templatesPath)) {
        File::deleteDirectory($this->templatesPath);
    }
});

function writeVersionTemplate(string $slug, array $content): void
{
    File::put(
        resource_path('plan-templates').'/'.$slug.'.yaml',
        Yaml::dump($content, 10)
    );
}

// =========================================================================
// PlanTemplateVersion model
// =========================================================================

describe('PlanTemplateVersion model', function () {
    it('creates a new version record on first use', function () {
        $content = ['name' => 'My Template', 'slug' => 'my-tpl', 'phases' => []];

        $version = PlanTemplateVersion::findOrCreateFromTemplate('my-tpl', $content);

        expect($version)->toBeInstanceOf(PlanTemplateVersion::class)
            ->and($version->slug)->toBe('my-tpl')
            ->and($version->version)->toBe(1)
            ->and($version->name)->toBe('My Template')
            ->and($version->content)->toBe($content)
            ->and($version->content_hash)->toBe(hash('sha256', json_encode($content, JSON_UNESCAPED_UNICODE)));
    });

    it('reuses existing version when content is identical', function () {
        $content = ['name' => 'Stable', 'slug' => 'stable', 'phases' => []];

        $v1 = PlanTemplateVersion::findOrCreateFromTemplate('stable', $content);
        $v2 = PlanTemplateVersion::findOrCreateFromTemplate('stable', $content);

        expect($v1->id)->toBe($v2->id)
            ->and(PlanTemplateVersion::where('slug', 'stable')->count())->toBe(1);
    });

    it('creates a new version when content changes', function () {
        $contentV1 = ['name' => 'Template', 'slug' => 'evolving', 'phases' => []];
        $contentV2 = ['name' => 'Template', 'slug' => 'evolving', 'phases' => [['name' => 'New Phase']]];

        $v1 = PlanTemplateVersion::findOrCreateFromTemplate('evolving', $contentV1);
        $v2 = PlanTemplateVersion::findOrCreateFromTemplate('evolving', $contentV2);

        expect($v1->id)->not->toBe($v2->id)
            ->and($v1->version)->toBe(1)
            ->and($v2->version)->toBe(2)
            ->and(PlanTemplateVersion::where('slug', 'evolving')->count())->toBe(2);
    });

    it('increments version numbers sequentially', function () {
        for ($i = 1; $i <= 3; $i++) {
            $content = ['name' => "Version {$i}", 'slug' => 'sequential', 'phases' => [$i]];
            $v = PlanTemplateVersion::findOrCreateFromTemplate('sequential', $content);
            expect($v->version)->toBe($i);
        }
    });

    it('scopes versions by slug independently', function () {
        $alpha = ['name' => 'Alpha', 'slug' => 'alpha', 'phases' => []];
        $beta = ['name' => 'Beta', 'slug' => 'beta', 'phases' => []];

        $vA = PlanTemplateVersion::findOrCreateFromTemplate('alpha', $alpha);
        $vB = PlanTemplateVersion::findOrCreateFromTemplate('beta', $beta);

        expect($vA->version)->toBe(1)
            ->and($vB->version)->toBe(1)
            ->and($vA->id)->not->toBe($vB->id);
    });

    it('returns history for a slug newest first', function () {
        $content1 = ['name' => 'T', 'slug' => 'hist', 'phases' => [1]];
        $content2 = ['name' => 'T', 'slug' => 'hist', 'phases' => [2]];

        PlanTemplateVersion::findOrCreateFromTemplate('hist', $content1);
        PlanTemplateVersion::findOrCreateFromTemplate('hist', $content2);

        $history = PlanTemplateVersion::historyFor('hist');

        expect($history)->toHaveCount(2)
            ->and($history[0]->version)->toBe(2)
            ->and($history[1]->version)->toBe(1);
    });

    it('returns empty collection when no versions exist for slug', function () {
        $history = PlanTemplateVersion::historyFor('nonexistent-slug');

        expect($history)->toBeEmpty();
    });
});

// =========================================================================
// Plan creation snapshots a template version
// =========================================================================

describe('plan creation version tracking', function () {
    it('creates a version record when creating a plan from a template', function () {
        writeVersionTemplate('versioned-plan', [
            'name' => 'Versioned Plan',
            'phases' => [['name' => 'Phase 1']],
        ]);

        $plan = $this->service->createPlan('versioned-plan', [], [], $this->workspace);

        expect($plan->template_version_id)->not->toBeNull()
            ->and(PlanTemplateVersion::where('slug', 'versioned-plan')->count())->toBe(1);
    });

    it('associates the plan with the template version', function () {
        writeVersionTemplate('linked-version', [
            'name' => 'Linked Version',
            'phases' => [],
        ]);

        $plan = $this->service->createPlan('linked-version', [], [], $this->workspace);

        expect($plan->templateVersion)->toBeInstanceOf(PlanTemplateVersion::class)
            ->and($plan->templateVersion->slug)->toBe('linked-version')
            ->and($plan->templateVersion->version)->toBe(1);
    });

    it('stores template version number in metadata', function () {
        writeVersionTemplate('meta-version', [
            'name' => 'Meta Version',
            'phases' => [],
        ]);

        $plan = $this->service->createPlan('meta-version', [], [], $this->workspace);

        expect($plan->metadata['template_version'])->toBe(1);
    });

    it('reuses the same version record for multiple plans from unchanged template', function () {
        writeVersionTemplate('shared-version', [
            'name' => 'Shared Template',
            'phases' => [],
        ]);

        $plan1 = $this->service->createPlan('shared-version', [], [], $this->workspace);
        $plan2 = $this->service->createPlan('shared-version', [], [], $this->workspace);

        expect($plan1->template_version_id)->toBe($plan2->template_version_id)
            ->and(PlanTemplateVersion::where('slug', 'shared-version')->count())->toBe(1);
    });

    it('creates a new version when template content changes between plan creations', function () {
        writeVersionTemplate('changing-template', [
            'name' => 'Original',
            'phases' => [],
        ]);

        $plan1 = $this->service->createPlan('changing-template', [], [], $this->workspace);

        // Simulate template file update
        writeVersionTemplate('changing-template', [
            'name' => 'Updated',
            'phases' => [['name' => 'Added Phase']],
        ]);

        // Re-resolve the service so it reads fresh YAML
        $freshService = new PlanTemplateService;
        $plan2 = $freshService->createPlan('changing-template', [], [], $this->workspace);

        expect($plan1->template_version_id)->not->toBe($plan2->template_version_id)
            ->and($plan1->templateVersion->version)->toBe(1)
            ->and($plan2->templateVersion->version)->toBe(2)
            ->and(PlanTemplateVersion::where('slug', 'changing-template')->count())->toBe(2);
    });

    it('existing plans keep their version reference after template update', function () {
        writeVersionTemplate('stable-ref', [
            'name' => 'Stable',
            'phases' => [],
        ]);

        $plan = $this->service->createPlan('stable-ref', [], [], $this->workspace);
        $originalVersionId = $plan->template_version_id;

        // Update template file
        writeVersionTemplate('stable-ref', [
            'name' => 'Stable Updated',
            'phases' => [['name' => 'New Phase']],
        ]);

        // Reload plan from DB - version reference must be unchanged
        $plan->refresh();

        expect($plan->template_version_id)->toBe($originalVersionId)
            ->and($plan->templateVersion->name)->toBe('Stable');
    });

    it('snapshots raw template content before variable substitution', function () {
        writeVersionTemplate('raw-snapshot', [
            'name' => '{{ project }} Plan',
            'phases' => [['name' => '{{ project }} Setup']],
        ]);

        $plan = $this->service->createPlan('raw-snapshot', ['project' => 'MyApp'], [], $this->workspace);

        // Version content should retain placeholders, not the substituted values
        $versionContent = $plan->templateVersion->content;

        expect($versionContent['name'])->toBe('{{ project }} Plan')
            ->and($versionContent['phases'][0]['name'])->toBe('{{ project }} Setup');
    });
});

// =========================================================================
// PlanTemplateService version methods
// =========================================================================

describe('PlanTemplateService version methods', function () {
    it('getVersionHistory returns empty array when no plans created', function () {
        $history = $this->service->getVersionHistory('no-plans-yet');

        expect($history)->toBeArray()->toBeEmpty();
    });

    it('getVersionHistory returns version summaries after plan creation', function () {
        writeVersionTemplate('hist-service', [
            'name' => 'History Template',
            'phases' => [],
        ]);

        $this->service->createPlan('hist-service', [], [], $this->workspace);

        $history = $this->service->getVersionHistory('hist-service');

        expect($history)->toHaveCount(1)
            ->and($history[0])->toHaveKeys(['id', 'slug', 'version', 'name', 'content_hash', 'created_at'])
            ->and($history[0]['slug'])->toBe('hist-service')
            ->and($history[0]['version'])->toBe(1)
            ->and($history[0]['name'])->toBe('History Template');
    });

    it('getVersionHistory orders newest version first', function () {
        writeVersionTemplate('order-hist', ['name' => 'V1', 'phases' => []]);
        $this->service->createPlan('order-hist', [], [], $this->workspace);

        writeVersionTemplate('order-hist', ['name' => 'V2', 'phases' => [['name' => 'p']]]);
        $freshService = new PlanTemplateService;
        $freshService->createPlan('order-hist', [], [], $this->workspace);

        $history = $this->service->getVersionHistory('order-hist');

        expect($history[0]['version'])->toBe(2)
            ->and($history[1]['version'])->toBe(1);
    });

    it('getVersion returns content for an existing version', function () {
        $templateContent = ['name' => 'Get Version Test', 'phases' => []];
        writeVersionTemplate('get-version', $templateContent);

        $this->service->createPlan('get-version', [], [], $this->workspace);

        $content = $this->service->getVersion('get-version', 1);

        expect($content)->not->toBeNull()
            ->and($content['name'])->toBe('Get Version Test');
    });

    it('getVersion returns null for nonexistent version', function () {
        $content = $this->service->getVersion('nonexistent', 99);

        expect($content)->toBeNull();
    });

    it('getVersion returns the correct historic content when template has changed', function () {
        writeVersionTemplate('historic-content', ['name' => 'Original Name', 'phases' => []]);
        $this->service->createPlan('historic-content', [], [], $this->workspace);

        writeVersionTemplate('historic-content', ['name' => 'Updated Name', 'phases' => [['name' => 'p']]]);
        $freshService = new PlanTemplateService;
        $freshService->createPlan('historic-content', [], [], $this->workspace);

        expect($this->service->getVersion('historic-content', 1)['name'])->toBe('Original Name')
            ->and($this->service->getVersion('historic-content', 2)['name'])->toBe('Updated Name');
    });
});
