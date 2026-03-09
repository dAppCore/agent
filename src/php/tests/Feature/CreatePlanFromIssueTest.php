<?php

/*
 * Core PHP Framework
 *
 * Licensed under the European Union Public Licence (EUPL) v1.2.
 * See LICENSE file for details.
 */

declare(strict_types=1);

use Core\Mod\Agentic\Actions\Forge\CreatePlanFromIssue;
use Core\Mod\Agentic\Models\AgentPlan;
use Core\Tenant\Models\Workspace;

beforeEach(function () {
    $this->workspace = Workspace::factory()->create();
});

it('creates a plan from a work item', function () {
    $workItem = [
        'epic_number' => 1,
        'issue_number' => 5,
        'issue_title' => 'Add colour picker component',
        'issue_body' => "## Requirements\n- [ ] Create picker UI\n- [ ] Add validation\n- [ ] Write tests",
        'assignee' => 'virgil',
        'repo_owner' => 'core',
        'repo_name' => 'app',
        'needs_coding' => true,
        'has_pr' => false,
    ];

    $plan = CreatePlanFromIssue::run($workItem, $this->workspace->id);

    expect($plan)->toBeInstanceOf(AgentPlan::class);
    expect($plan->title)->toBe('Add colour picker component');
    expect($plan->slug)->toBe('forge-core-app-5');
    expect($plan->status)->toBe(AgentPlan::STATUS_DRAFT);
    expect($plan->metadata)->toMatchArray([
        'source' => 'forgejo',
        'epic_number' => 1,
        'issue_number' => 5,
        'repo_owner' => 'core',
        'repo_name' => 'app',
        'assignee' => 'virgil',
    ]);

    // Verify phases and tasks
    expect($plan->agentPhases)->toHaveCount(1);
    $phase = $plan->agentPhases->first();
    expect($phase->name)->toBe('Resolve issue #5');
    expect($phase->tasks)->toHaveCount(3);
    expect($phase->tasks[0]['name'])->toBe('Create picker UI');
    expect($phase->tasks[1]['name'])->toBe('Add validation');
    expect($phase->tasks[2]['name'])->toBe('Write tests');
});

it('creates a plan with no tasks if body has no checklist', function () {
    $workItem = [
        'epic_number' => 1,
        'issue_number' => 7,
        'issue_title' => 'Investigate performance regression',
        'issue_body' => 'The dashboard is slow. Please investigate.',
        'assignee' => null,
        'repo_owner' => 'core',
        'repo_name' => 'app',
        'needs_coding' => true,
        'has_pr' => false,
    ];

    $plan = CreatePlanFromIssue::run($workItem, $this->workspace->id);

    expect($plan)->toBeInstanceOf(AgentPlan::class);
    expect($plan->title)->toBe('Investigate performance regression');
    expect($plan->agentPhases)->toHaveCount(1);
    expect($plan->agentPhases->first()->tasks)->toBeEmpty();
});

it('skips duplicate plans for same issue', function () {
    $workItem = [
        'epic_number' => 1,
        'issue_number' => 9,
        'issue_title' => 'Fix the widget',
        'issue_body' => "- [ ] Do the thing",
        'assignee' => null,
        'repo_owner' => 'core',
        'repo_name' => 'app',
        'needs_coding' => true,
        'has_pr' => false,
    ];

    $first = CreatePlanFromIssue::run($workItem, $this->workspace->id);
    $second = CreatePlanFromIssue::run($workItem, $this->workspace->id);

    expect($second->id)->toBe($first->id);
    expect(AgentPlan::where('slug', 'forge-core-app-9')->count())->toBe(1);
});
