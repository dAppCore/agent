<?php

/**
 * UseCase: Agentic Admin Panel (Basic Flow)
 *
 * Acceptance test for the Agentic admin panel.
 * Tests the primary admin CRUD flow through templates, sessions, and plans.
 *
 * Uses translation keys to get expected values - tests won't break on copy changes.
 */

use Core\Mod\Agentic\Models\AgentPlan;
use Core\Tenant\Models\User;
use Core\Tenant\Models\Workspace;

describe('Agentic Admin Panel', function () {
    beforeEach(function () {
        // Create user with workspace
        $this->user = User::factory()->create([
            'email' => 'test@example.com',
            'password' => bcrypt('password'),
        ]);

        $this->workspace = Workspace::factory()->create();
        $this->workspace->users()->attach($this->user->id, [
            'role' => 'owner',
            'is_default' => true,
        ]);
    });

    it('can view the dashboard with all sections', function () {
        // Login
        $page = visit('/login');

        $page->fill('email', 'test@example.com')
            ->fill('password', 'password')
            ->click(__('pages::pages.login.submit'))
            ->assertPathContains('/hub');

        // Navigate to Agentic dashboard
        $page->navigate('/hub/agents')
            ->assertSee(__('agentic::agentic.dashboard.title'))
            ->assertSee(__('agentic::agentic.dashboard.subtitle'))
            ->assertSee(__('agentic::agentic.actions.refresh'))
            ->assertSee(__('agentic::agentic.dashboard.recent_activity'))
            ->assertSee(__('agentic::agentic.dashboard.top_tools'));
    });

    it('can view plans list with filters', function () {
        // Create a test plan
        $plan = AgentPlan::factory()->create([
            'workspace_id' => $this->workspace->id,
            'title' => 'Test Plan',
            'status' => 'active',
        ]);

        // Login
        $page = visit('/login');

        $page->fill('email', 'test@example.com')
            ->fill('password', 'password')
            ->click(__('pages::pages.login.submit'))
            ->assertPathContains('/hub');

        // Navigate to plans
        $page->navigate('/hub/agents/plans')
            ->assertSee(__('agentic::agentic.plans.title'))
            ->assertSee(__('agentic::agentic.plans.subtitle'))
            ->assertSee(__('agentic::agentic.plans.search_placeholder'))
            ->assertSee(__('agentic::agentic.filters.all_statuses'))
            ->assertSee(__('agentic::agentic.filters.all_workspaces'))
            ->assertSee(__('agentic::agentic.table.plan'))
            ->assertSee(__('agentic::agentic.table.workspace'))
            ->assertSee(__('agentic::agentic.table.status'))
            ->assertSee(__('agentic::agentic.table.progress'))
            ->assertSee(__('agentic::agentic.table.sessions'))
            ->assertSee(__('agentic::agentic.table.last_activity'))
            ->assertSee(__('agentic::agentic.table.actions'));
    });

    it('can view sessions list with filters', function () {
        // Login
        $page = visit('/login');

        $page->fill('email', 'test@example.com')
            ->fill('password', 'password')
            ->click(__('pages::pages.login.submit'))
            ->assertPathContains('/hub');

        // Navigate to sessions
        $page->navigate('/hub/agents/sessions')
            ->assertSee(__('agentic::agentic.sessions.title'))
            ->assertSee(__('agentic::agentic.sessions.subtitle'))
            ->assertSee(__('agentic::agentic.sessions.search_placeholder'))
            ->assertSee(__('agentic::agentic.filters.all_statuses'))
            ->assertSee(__('agentic::agentic.filters.all_agents'))
            ->assertSee(__('agentic::agentic.filters.all_workspaces'))
            ->assertSee(__('agentic::agentic.filters.all_plans'))
            ->assertSee(__('agentic::agentic.table.session'))
            ->assertSee(__('agentic::agentic.table.agent'))
            ->assertSee(__('agentic::agentic.table.duration'))
            ->assertSee(__('agentic::agentic.table.activity'));
    });

    it('can view templates page', function () {
        // Login
        $page = visit('/login');

        $page->fill('email', 'test@example.com')
            ->fill('password', 'password')
            ->click(__('pages::pages.login.submit'))
            ->assertPathContains('/hub');

        // Navigate to templates
        $page->navigate('/hub/agents/templates')
            ->assertSee(__('agentic::agentic.templates.title'))
            ->assertSee(__('agentic::agentic.templates.subtitle'))
            ->assertSee(__('agentic::agentic.actions.import'))
            ->assertSee(__('agentic::agentic.actions.back_to_plans'))
            ->assertSee(__('agentic::agentic.templates.stats.templates'))
            ->assertSee(__('agentic::agentic.templates.stats.categories'))
            ->assertSee(__('agentic::agentic.templates.stats.total_phases'))
            ->assertSee(__('agentic::agentic.templates.stats.with_variables'))
            ->assertSee(__('agentic::agentic.templates.search_placeholder'))
            ->assertSee(__('agentic::agentic.filters.all_categories'));
    });

    it('can view API keys page', function () {
        // Login
        $page = visit('/login');

        $page->fill('email', 'test@example.com')
            ->fill('password', 'password')
            ->click(__('pages::pages.login.submit'))
            ->assertPathContains('/hub');

        // Navigate to API keys
        $page->navigate('/hub/agents/api-keys')
            ->assertSee(__('agentic::agentic.api_keys.title'))
            ->assertSee(__('agentic::agentic.api_keys.subtitle'))
            ->assertSee(__('agentic::agentic.actions.create_key'))
            ->assertSee(__('agentic::agentic.api_keys.stats.total_keys'))
            ->assertSee(__('agentic::agentic.api_keys.stats.active'))
            ->assertSee(__('agentic::agentic.api_keys.stats.revoked'))
            ->assertSee(__('agentic::agentic.api_keys.stats.total_calls'))
            ->assertSee(__('agentic::agentic.table.name'))
            ->assertSee(__('agentic::agentic.table.permissions'))
            ->assertSee(__('agentic::agentic.table.rate_limit'))
            ->assertSee(__('agentic::agentic.table.usage'))
            ->assertSee(__('agentic::agentic.table.last_used'))
            ->assertSee(__('agentic::agentic.table.created'));
    });

    it('can view tool analytics page', function () {
        // Login
        $page = visit('/login');

        $page->fill('email', 'test@example.com')
            ->fill('password', 'password')
            ->click(__('pages::pages.login.submit'))
            ->assertPathContains('/hub');

        // Navigate to tool analytics
        $page->navigate('/hub/agents/tools')
            ->assertSee(__('agentic::agentic.tools.title'))
            ->assertSee(__('agentic::agentic.tools.subtitle'))
            ->assertSee(__('agentic::agentic.actions.view_all_calls'))
            ->assertSee(__('agentic::agentic.tools.stats.total_calls'))
            ->assertSee(__('agentic::agentic.tools.stats.successful'))
            ->assertSee(__('agentic::agentic.tools.stats.errors'))
            ->assertSee(__('agentic::agentic.tools.stats.success_rate'))
            ->assertSee(__('agentic::agentic.tools.stats.unique_tools'))
            ->assertSee(__('agentic::agentic.tools.daily_trend'))
            ->assertSee(__('agentic::agentic.tools.server_breakdown'))
            ->assertSee(__('agentic::agentic.tools.top_tools'));
    });

    it('can view tool calls page', function () {
        // Login
        $page = visit('/login');

        $page->fill('email', 'test@example.com')
            ->fill('password', 'password')
            ->click(__('pages::pages.login.submit'))
            ->assertPathContains('/hub');

        // Navigate to tool calls
        $page->navigate('/hub/agents/tools/calls')
            ->assertSee(__('agentic::agentic.tool_calls.title'))
            ->assertSee(__('agentic::agentic.tool_calls.subtitle'))
            ->assertSee(__('agentic::agentic.tool_calls.search_placeholder'))
            ->assertSee(__('agentic::agentic.filters.all_servers'))
            ->assertSee(__('agentic::agentic.filters.all_tools'))
            ->assertSee(__('agentic::agentic.filters.all_status'))
            ->assertSee(__('agentic::agentic.table.tool'))
            ->assertSee(__('agentic::agentic.table.server'))
            ->assertSee(__('agentic::agentic.table.time'));
    });

    it('shows empty state when no plans exist', function () {
        // Login
        $page = visit('/login');

        $page->fill('email', 'test@example.com')
            ->fill('password', 'password')
            ->click(__('pages::pages.login.submit'))
            ->assertPathContains('/hub');

        // Navigate to plans (should be empty)
        $page->navigate('/hub/agents/plans')
            ->assertSee(__('agentic::agentic.empty.no_plans'))
            ->assertSee(__('agentic::agentic.empty.plans_appear'));
    });

    it('shows empty state when no sessions exist', function () {
        // Login
        $page = visit('/login');

        $page->fill('email', 'test@example.com')
            ->fill('password', 'password')
            ->click(__('pages::pages.login.submit'))
            ->assertPathContains('/hub');

        // Navigate to sessions (should be empty)
        $page->navigate('/hub/agents/sessions')
            ->assertSee(__('agentic::agentic.empty.no_sessions'))
            ->assertSee(__('agentic::agentic.empty.sessions_appear'));
    });

    it('can clear filters', function () {
        // Login
        $page = visit('/login');

        $page->fill('email', 'test@example.com')
            ->fill('password', 'password')
            ->click(__('pages::pages.login.submit'))
            ->assertPathContains('/hub');

        // Navigate to plans and use filter
        $page->navigate('/hub/agents/plans')
            ->assertSee(__('agentic::agentic.plans.title'));

        // Type in search to trigger filter
        $page->type('input[placeholder="'.__('agentic::agentic.plans.search_placeholder').'"]', 'test')
            ->wait(1)
            ->assertSee(__('agentic::agentic.actions.clear'));

        $page->click(__('agentic::agentic.actions.clear'))
            ->wait(1);
    });
});
